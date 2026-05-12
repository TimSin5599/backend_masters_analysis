//go:build e2e

package e2e

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"github.com/testcontainers/testcontainers-go/wait"

	v1 "auth-service/internal/controller/http/v1"
	pgrepo "auth-service/internal/repository/postgres"
	redisrepo "auth-service/internal/repository/redis"
	"auth-service/internal/usecase"
	"auth-service/pkg/postgres"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"
)

// srvURL — адрес тестового HTTP-сервера, доступен всем тестам пакета.
var srvURL string

const testJWTKey = "e2e-test-jwt-secret-key"

func TestMain(m *testing.M) {
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	// 1. Postgres-контейнер
	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:15-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.WithInitScripts("../../deploy/init_schema.sql"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2),
		),
	)
	if err != nil {
		panic("testcontainers postgres: " + err.Error())
	}
	defer pgContainer.Terminate(ctx) //nolint:errcheck

	// 2. Redis-контейнер
	redisContainer, err := tcredis.Run(ctx, "redis:7-alpine")
	if err != nil {
		panic("testcontainers redis: " + err.Error())
	}
	defer redisContainer.Terminate(ctx) //nolint:errcheck

	// 3. Строки подключения
	pgDSN, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("pgContainer.ConnectionString: " + err.Error())
	}

	redisAddr, err := redisContainer.Endpoint(ctx, "")
	if err != nil {
		panic("redisContainer.Endpoint: " + err.Error())
	}

	// 4. Клиенты
	pool, err := pgxpool.New(ctx, pgDSN)
	if err != nil {
		panic("pgxpool.New: " + err.Error())
	}
	defer pool.Close()

	redisClient := goredis.NewClient(&goredis.Options{Addr: redisAddr})
	defer redisClient.Close() //nolint:errcheck

	// 5. Репозитории и UseCase
	pgRepo := pgrepo.NewPGRepo(&postgres.Postgres{Pool: pool})
	redisRepo := redisrepo.NewRedisRepo(redisClient)

	authUC := usecase.NewAuth(pgRepo, redisRepo, testJWTKey)
	userUC := usecase.NewUser(pgRepo, redisRepo)

	// 6. Роутер
	handler := gin.New()
	v1.NewRouter(handler, authUC, userUC, testJWTKey, "")

	// 7. httptest.Server
	srv := httptest.NewServer(handler)
	defer srv.Close()
	srvURL = srv.URL

	os.Exit(m.Run())
}
