//go:build e2e

// E2E тесты для manage-service с Testcontainers.
// Запуск: go test -tags=e2e ./e2e/... -v -timeout 120s
// Требуют запущенный Docker.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func doRequest(t *testing.T, method, path string, body interface{}) *http.Response {
	t.Helper()
	var buf *bytes.Buffer
	if body != nil {
		b, _ := json.Marshal(body)
		buf = bytes.NewBuffer(b)
	} else {
		buf = &bytes.Buffer{}
	}
	req, err := http.NewRequest(method, srvURL+path, buf)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func post(t *testing.T, path string, body interface{}) *http.Response {
	return doRequest(t, http.MethodPost, path, body)
}

func get(t *testing.T, path string) *http.Response {
	return doRequest(t, http.MethodGet, path, nil)
}

func del(t *testing.T, path string) *http.Response {
	return doRequest(t, http.MethodDelete, path, nil)
}

// TestE2E_ApplicantLifecycle тестирует создание, получение и удаление абитуриента.
func TestE2E_ApplicantLifecycle(t *testing.T) {
	// Для создания абитуриента нужна программа — убедимся, что она есть в БД (seed создаёт её через init_schema)
	// Если сид не создаёт программу — создадим её.
	var programID int64 = 1

	// Попробуем создать программу, чтобы получить валидный program_id
	t.Run("EnsureProgram", func(t *testing.T) {
		resp := post(t, "/v1/programs", map[string]interface{}{
			"title":       "Тестовая программа E2E",
			"year":        2024,
			"description": "E2E test",
		})
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusCreated {
			var body map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&body)
			if id, ok := body["id"].(float64); ok {
				programID = int64(id)
			}
		}
		// 400/422 тоже ок — значит программа уже есть или эндпоинт требует другие поля
	})

	ts := time.Now().UnixNano()
	firstName := fmt.Sprintf("E2E%d", ts)

	var applicantID float64
	t.Run("CreateApplicant", func(t *testing.T) {
		resp := post(t, "/v1/applicants", map[string]interface{}{
			"program_id": programID,
			"first_name": firstName,
			"last_name":  "Тестов",
			"patronymic": "Тестович",
		})
		defer resp.Body.Close()
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		id, ok := body["id"].(float64)
		require.True(t, ok, "expected numeric id in response, got: %v", body)
		require.Greater(t, id, float64(0))
		applicantID = id
	})

	if applicantID == 0 {
		t.Skip("applicant creation failed, skipping further steps")
	}
	idStr := fmt.Sprintf("%.0f", applicantID)

	t.Run("ListApplicants", func(t *testing.T) {
		resp := get(t, fmt.Sprintf("/v1/applicants?program_id=%d", programID))
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body []map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.GreaterOrEqual(t, len(body), 1)
	})

	t.Run("GetApplicantData", func(t *testing.T) {
		resp := get(t, fmt.Sprintf("/v1/applicants/%s/data?category=work", idStr))
		defer resp.Body.Close()
		assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNotFound,
			"unexpected status %d", resp.StatusCode)
	})

	t.Run("ListDocuments", func(t *testing.T) {
		resp := get(t, fmt.Sprintf("/v1/applicants/%s/documents", idStr))
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GetEvaluationCriteria", func(t *testing.T) {
		resp := get(t, fmt.Sprintf("/v1/applicants/%s/criteria", idStr))
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("DeleteApplicant", func(t *testing.T) {
		resp := del(t, fmt.Sprintf("/v1/applicants/%s", idStr))
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestE2E_Criteria тестирует CRUD критериев оценки.
func TestE2E_Criteria(t *testing.T) {
	code := fmt.Sprintf("E2E_%d", time.Now().UnixNano())

	t.Run("ListCriteria", func(t *testing.T) {
		resp := get(t, "/v1/criteria")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var body []interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		// init_schema.sql засевает 9 критериев
		assert.GreaterOrEqual(t, len(body), 9)
	})

	t.Run("CreateCriteria", func(t *testing.T) {
		resp := post(t, "/v1/criteria", map[string]interface{}{
			"code":           code,
			"title":          "E2E Критерий",
			"max_score":      10,
			"type":           "BASE",
			"scheme":         "default",
			"document_types": []string{},
		})
		defer resp.Body.Close()
		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})

	t.Run("DeleteCriteria", func(t *testing.T) {
		resp := doRequest(t, http.MethodDelete, fmt.Sprintf("/v1/criteria/%s", code), nil)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestE2E_InvalidApplicantID проверяет 400 при нечисловом id.
func TestE2E_InvalidApplicantID(t *testing.T) {
	resp := get(t, "/v1/applicants/not-a-number/documents")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

// TestE2E_DocumentStatus_NotFound проверяет 404 для несуществующего документа.
func TestE2E_DocumentStatus_NotFound(t *testing.T) {
	resp := get(t, "/v1/documents/99999999/status")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}
