package main

import (
	"log"

	"statistics-service/config"
	"statistics-service/pkg/httpserver"
	"statistics-service/pkg/logger"
	"statistics-service/pkg/postgres"

	v1 "statistics-service/internal/controller/http/v1"

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

	// HTTP Server
	handler := gin.New()

	// 3. Router
	v1.NewRouter(handler)

	httpServer := httpserver.New(handler, cfg.HTTP.Port)
	l.Info("app - Run - Server started on port: " + cfg.HTTP.Port)

	// Waiting signal
	select {
	case err = <-httpServer.Notify():
		l.Error("app - Run - httpServer.Notify: %v", err)
	}
}
