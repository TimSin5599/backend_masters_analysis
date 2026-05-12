//go:build e2e

package e2e

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	v1 "statistics-service/internal/controller/http/v1"
	"statistics-service/internal/repository"
	"statistics-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// srvURL — адрес тестового HTTP-сервера, доступен всем тестам пакета.
var srvURL string

func TestMain(m *testing.M) {
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	// 1. Поднять Postgres-контейнер
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

	// 2. Получить строку подключения
	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("pgContainer.ConnectionString: " + err.Error())
	}

	// 3. Создать pgxpool
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic("pgxpool.New: " + err.Error())
	}
	defer pool.Close()

	// 4. Собрать граф зависимостей сервиса
	statsRepo := repository.New(pool)
	statsUC := usecase.New(statsRepo)

	handler := gin.New()
	v1.NewRouter(handler, statsUC, "")

	// 5. Запустить httptest.Server
	srv := httptest.NewServer(handler)
	defer srv.Close()
	srvURL = srv.URL

	os.Exit(m.Run())
}
