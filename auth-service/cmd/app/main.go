package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"auth-service/config"
	authDocs "auth-service/docs"
	v1 "auth-service/internal/controller/http/v1"
	pgrepo "auth-service/internal/repository/postgres"
	redisrepo "auth-service/internal/repository/redis"
	"auth-service/internal/usecase"
	"auth-service/pkg/httpserver"
	"auth-service/pkg/logger"
	"auth-service/pkg/postgres"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// @title           Auth Service API
// @version         1.0
// @description     Authentication service
// @host            localhost:8081
// @BasePath        /

func main() {
	// Configuration
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Config error: %s", err)
	}

	// Swagger
	swaggerHost := cfg.Swagger.Host
	if swaggerHost == "" {
		swaggerHost = "localhost:" + cfg.HTTP.Port
	}
	authDocs.SwaggerInfo.Title = cfg.App.Name
	authDocs.SwaggerInfo.Version = cfg.App.Version
	authDocs.SwaggerInfo.Host = swaggerHost

	// Logger
	l := logger.New(cfg.Log.Level)

	// PostgreSQL
	pg, err := postgres.New(cfg.PG.URL, postgres.MaxPoolSize(cfg.PG.PoolMax))
	if err != nil {
		l.Fatal("app - Run - postgres.New: %w", err)
	}
	defer pg.Close()

	// Redis
	redisOpts, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		l.Fatal("app - Run - redis.ParseURL: %w", err)
	}
	redisClient := redis.NewClient(redisOpts)
	defer func() {
		_ = redisClient.Close()
	}()

	// Ping Redis для проверки соединения
	if err = redisClient.Ping(context.Background()).Err(); err != nil {
		l.Fatal("app - Run - redisClient.Ping: %w", err)
	}
	l.Info("%s", "app - Run - Redis connected: "+cfg.Redis.URL)

	// Repositories
	pgRepo := pgrepo.NewPGRepo(pg)
	redisRepo := redisrepo.NewRedisRepo(redisClient)

	// Use Case
	authUseCase := usecase.NewAuth(pgRepo, redisRepo, cfg.JWT.SignKey)
	userUseCase := usecase.NewUser(pgRepo, redisRepo)

	// HTTP Server
	handler := gin.New()
	handler.Use(gin.Logger())
	handler.Use(gin.Recovery())
	v1.NewRouter(handler, authUseCase, userUseCase, cfg.JWT.SignKey, cfg.CORS.AllowOrigin)
	httpServer := httpserver.New(handler, cfg.HTTP.Port)
	l.Info("%s", "app - Run - Server started on port: "+cfg.HTTP.Port)

	// Waiting signal
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		l.Info("%s", "app - Run - signal: "+s.String())
	case err = <-httpServer.Notify():
		l.Error("app - Run - httpServer.Notify: %w", err)
	}

	// Shutdown
	if err = httpServer.Shutdown(); err != nil {
		l.Error("app - Run - httpServer.Shutdown: %w", err)
	}
}
