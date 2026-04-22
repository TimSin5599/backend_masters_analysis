package main

import (
	"context"
	"log"
	"time"

	"manage-service/config"
	"manage-service/pkg/httpserver"
	"manage-service/pkg/logger"
	"manage-service/pkg/postgres"

	v1 "manage-service/internal/controller/http/v1"
	"manage-service/internal/infrastructure/extraction"
	"manage-service/internal/infrastructure/s3"
	"manage-service/internal/rabbitmq"
	"manage-service/internal/repository"
	"manage-service/internal/usecase"
	ws "manage-service/internal/websocket"

	"github.com/gin-gonic/gin"
	"os"
	"os/signal"
	"syscall"
)

// @title           Manage Service API
// @version         1.0
// @description     API for managing applicants and documents
// @host            localhost:8080

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Введите токен в формате: Bearer {token}
func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Logger
	l := logger.New(cfg.Log.Level)

	// Postgres
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal("app - Run - postgres.New: %w", err)
	}
	defer pg.Close()

	// 1. Repository
	appRepo := repository.NewApplicantRepo(pg.Pool)
	docRepo := repository.NewDocumentRepo(pg.Pool)
	expertRepo := repository.NewExpertRepo(pg.Pool)
	progRepo := repository.NewProgramRepo(pg.Pool)
	queueRepo := repository.NewDocumentQueueRepo(pg.Pool)

	// 2. Infrastructure
	// MinIO
	minioClient, err := s3.New(cfg.MinIO.Endpoint, cfg.MinIO.AccessKey, cfg.MinIO.SecretKey, cfg.MinIO.Bucket, false)
	if err != nil {
		l.Error("app - Run - s3.New: %v", err)
		// Don't fail hard for now, allowing partial start if minio is down, but functionality will break
	}

	// RabbitMQ Producer
	producer, err := rabbitmq.NewProducer(cfg.RabbitMQ.URL)
	if err != nil {
		l.Fatal("app - Run - rabbitmq.NewProducer: %v", err)
	}
	defer producer.Close()

	// Extraction Service
	extractionClient := extraction.New(cfg.Extraction.ServiceURL)

	// 3. UseCase
	// Scoring client для AI-оценивания портфолио (POST /v1/score на data-extraction-service)
	scoringClient := extraction.NewScoringClient(cfg.Extraction.ServiceURL)

	docUC := usecase.NewDocumentUseCase(docRepo, appRepo, queueRepo, producer, extractionClient, minioClient)
	appUC := usecase.NewApplicantUseCase(appRepo, docRepo, docUC, expertRepo)
	expertUC := usecase.NewExpertUseCase(expertRepo, appRepo, scoringClient)
	progUC := usecase.NewProgramUseCase(progRepo)

	// Подключаем AI-оценивание к applicantUC (разрыв циклической зависимости через интерфейс)
	appUC.SetAIScoringTrigger(expertUC)

	// 4. WebSocket Hub
	hub := ws.NewHub()

	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 5. RabbitMQ Consumer
	consumer, err := rabbitmq.NewConsumer(cfg.RabbitMQ.URL, appRepo, queueRepo, appUC, docUC, docRepo, minioClient, extractionClient, hub)
	if err != nil {
		l.Fatal("app - Run - rabbitmq.NewConsumer: %v", err)
	}
	defer consumer.Close()
	// Recovery worker: re-enqueues tasks stuck in "processing"/"pending" after service crashes.
	// Runs 30s after startup, then every 10 minutes. Threshold: 20min > consumer's 12min timeout.
	go rabbitmq.StartPeriodicRecovery(ctx, queueRepo, docRepo, producer, 10*time.Minute, 20*time.Minute)

	// Supervision loop: restart consumer goroutine if it exits unexpectedly (e.g. after a panic)
	go func() {
		for {
			if err := consumer.Start(ctx); err != nil {
				l.Error("app - consumer.Start: %v", err)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(5 * time.Second):
				l.Info("app - restarting RabbitMQ consumer...")
			}
		}
	}()

	// HTTP Server
	handler := gin.New()
	handler.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "manage-service"})
	})

	// Router
	v1.NewRouter(handler, appUC, docUC, expertUC, progUC, hub, cfg.JWT.SignKey, cfg.CORS.AllowOrigin)

	httpServer := httpserver.New(handler, cfg.HTTP.Port)
	l.Info("app - Run - Server started on port: %s", cfg.HTTP.Port)

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		l.Error("app - Run - httpServer.Notify: %v", err)
	}

	// Shutdown
	cancel() // Stop consumer and other workers
	err = httpServer.Shutdown()
	if err != nil {
		l.Error("app - Run - httpServer.Shutdown: %v", err)
	}
}
