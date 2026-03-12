package main

import (
	"auth-service/config"
	v1 "auth-service/internal/controller/http/v1"
	"auth-service/internal/repository"
	"auth-service/internal/usecase"
	"auth-service/pkg/httpserver"
	"auth-service/pkg/logger"
	"auth-service/pkg/postgres"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Repository
	repo := repository.NewPGRepo(pg)

	// Use Case
	authUseCase := usecase.New(repo, cfg.JWT.SignKey, 24*time.Hour)

	// HTTP Server
	handler := gin.New()
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())
	v1.NewRouter(handler, authUseCase, cfg.JWT.SignKey)
	httpServer := httpserver.New(handler, cfg.HTTP.Port)
	l.Info("app - Run - Server started on port: " + cfg.HTTP.Port)

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("app - Run - signal: " + s.String())
	case err = <-httpServer.Notify():
		l.Error("app - Run - httpServer.Notify: %w", err)
	}

	// Shutdown
	err = httpServer.Shutdown()
	if err != nil {
		l.Error("app - Run - httpServer.Shutdown: %w", err)
	}
}
