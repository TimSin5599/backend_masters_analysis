package main

import (
	"log"

	"statistics-service/config"
	v1 "statistics-service/internal/controller/http/v1"
	"statistics-service/internal/repository"
	"statistics-service/internal/usecase"
	"statistics-service/pkg/httpserver"
	"statistics-service/pkg/logger"
	"statistics-service/pkg/postgres"

	_ "statistics-service/docs"

	"github.com/gin-gonic/gin"
)

// @title           Statistics Service API
// @version         1.0
// @description     API статистики приёмной кампании: метрики по абитуриентам, обработке ИИ и динамике по периодам.
// @host            localhost:8083
// @BasePath        /
func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	l := logger.New(cfg.Log.Level)

	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal("app - Run - postgres.New: %w", err)
	}
	defer pg.Close()

	statsRepo := repository.New(pg.Pool)
	statsUC := usecase.New(statsRepo)

	handler := gin.New()
	v1.NewRouter(handler, statsUC)

	httpServer := httpserver.New(handler, cfg.HTTP.Port)
	l.Info("app - Run - Server started on port: " + cfg.HTTP.Port)

	select {
	case err = <-httpServer.Notify():
		l.Error("app - Run - httpServer.Notify: %v", err)
	}
}
