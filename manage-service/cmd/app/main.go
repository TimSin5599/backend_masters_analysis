package main

import (
	"context"
	"log"

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
)

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
	repo := repository.NewApplicantRepo(pg.Pool)
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
	u := usecase.New(repo, queueRepo, producer, extractionClient, minioClient)

	// 4. WebSocket Hub
	hub := ws.NewHub()

	// 5. RabbitMQ Consumer
	consumer, err := rabbitmq.NewConsumer(cfg.RabbitMQ.URL, repo, queueRepo, u, minioClient, extractionClient, hub)
	if err != nil {
		l.Fatal("app - Run - rabbitmq.NewConsumer: %v", err)
	}
	defer consumer.Close()
	go consumer.Start(context.Background())

	// HTTP Server
	handler := gin.New()
	handler.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "manage-service"})
	})

	// Router
	v1.NewRouter(handler, u, queueRepo, hub)

	httpServer := httpserver.New(handler, cfg.HTTP.Port)
	l.Info("app - Run - Server started on port: " + cfg.HTTP.Port)

	// Waiting signal
	select {
	case err = <-httpServer.Notify():
		l.Error("app - Run - httpServer.Notify: %v", err)
	}
}
