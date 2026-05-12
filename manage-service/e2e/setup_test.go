//go:build e2e

package e2e

import (
	"context"
	"errors"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	v1 "manage-service/internal/controller/http/v1"
	"manage-service/internal/domain/entity"
	"manage-service/internal/repository"
	"manage-service/internal/sse"
	"manage-service/internal/usecase"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
)

// srvURL — адрес тестового HTTP-сервера, доступен всем тестам пакета.
var srvURL string

// authToken — Bearer-токен для запросов к защищённым эндпоинтам.
var authToken string

const testJWTKey = "e2e-test-secret"

// makeAdminToken генерирует JWT с ролью admin, подписанный testJWTKey.
func makeAdminToken() string {
	claims := jwt.MapClaims{
		"sub": "e2e-admin-user",
		"user": map[string]interface{}{
			"roles": []string{"admin"},
		},
		"exp": time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTKey))
	if err != nil {
		panic("makeAdminToken: " + err.Error())
	}
	return signed
}

// ─── Заглушки для инфраструктурных зависимостей ──────────────────────────────

type stubS3 struct{}

func (s *stubS3) UploadFile(_ context.Context, _ string, _ []byte) error {
	return errors.New("s3 not available in e2e tests")
}
func (s *stubS3) GetFile(_ context.Context, _ string) ([]byte, error) {
	return nil, errors.New("s3 not available in e2e tests")
}
func (s *stubS3) DeleteFile(_ context.Context, _ string) error {
	return errors.New("s3 not available in e2e tests")
}
func (s *stubS3) ListFiles(_ context.Context, _ string) ([]string, error) {
	return nil, errors.New("s3 not available in e2e tests")
}
func (s *stubS3) CopyFile(_ context.Context, _, _ string) error {
	return errors.New("s3 not available in e2e tests")
}

type stubProducer struct{}

func (s *stubProducer) PublishTask(_ entity.DocumentQueueTask) error {
	return errors.New("rabbitmq not available in e2e tests")
}

type stubExtraction struct{}

func (s *stubExtraction) TriggerExtraction(_ context.Context, _ entity.Document, _ []byte) (map[string]string, error) {
	return nil, errors.New("extraction not available in e2e tests")
}
func (s *stubExtraction) ClassifyDocument(_ context.Context, _ string, _ []byte) (string, []string, error) {
	return "", nil, errors.New("extraction not available in e2e tests")
}
func (s *stubExtraction) GenerateAnnotation(_ context.Context, _ map[string]interface{}) (string, error) {
	return "", errors.New("extraction not available in e2e tests")
}

// ─── TestMain ─────────────────────────────────────────────────────────────────

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

	// 2. Строка подключения
	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		panic("pgContainer.ConnectionString: " + err.Error())
	}

	// 3. pgxpool
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		panic("pgxpool.New: " + err.Error())
	}
	defer pool.Close()

	// 4. Репозитории
	appRepo := repository.NewApplicantRepo(pool)
	docRepo := repository.NewDocumentRepo(pool)
	expertRepo := repository.NewExpertRepo(pool)
	progRepo := repository.NewProgramRepo(pool)
	queueRepo := repository.NewDocumentQueueRepo(pool)

	// 5. UseCases (MinIO / RabbitMQ / Extraction — заглушки)
	docUC := usecase.NewDocumentUseCase(docRepo, appRepo, queueRepo, &stubProducer{}, &stubExtraction{}, &stubS3{})
	appUC := usecase.NewApplicantUseCase(appRepo, docRepo, docUC, expertRepo)
	expertUC := usecase.NewExpertUseCase(expertRepo, appRepo, nil)
	progUC := usecase.NewProgramUseCase(progRepo)
	appUC.SetAIScoringTrigger(expertUC)

	// 6. SSE Hub (не запускаем Consumer — он требует RabbitMQ)
	hub := sse.NewHub()

	// 7. Gin-роутер
	handler := gin.New()
	const testJWTKey = "e2e-test-secret"
	v1.NewRouter(handler, appUC, docUC, expertUC, progUC, hub, testJWTKey, "", nil)

	// 8. httptest.Server
	srv := httptest.NewServer(handler)
	defer srv.Close()
	srvURL = srv.URL

	// 9. Сгенерировать токен для тестов
	authToken = makeAdminToken()

	os.Exit(m.Run())
}
