//go:build e2e

// E2E тесты для statistics-service с Testcontainers.
// Запуск: go test -tags=e2e ./e2e/... -v -timeout 120s
// Требуют запущенный Docker.
package e2e

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func httpGet(t *testing.T, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srvURL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// TestE2E_Overview проверяет /v1/stats/overview.
func TestE2E_Overview(t *testing.T) {
	t.Run("NoFilter", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/overview")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		_, ok := body["total_applicants"]
		assert.True(t, ok, "response should contain total_applicants")
	})

	t.Run("WithProgramID", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/overview?program_id=1")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}

// TestE2E_Dynamics проверяет /v1/stats/dynamics.
func TestE2E_Dynamics(t *testing.T) {
	t.Run("DefaultPeriod", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/dynamics")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body []interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.NotNil(t, body)
	})

	t.Run("WeeklyPeriod", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/dynamics?period=weekly")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("MonthlyPeriod", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/dynamics?period=monthly")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("InvalidPeriodFallsBack", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/dynamics?period=hourly")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("WithProgramID", func(t *testing.T) {
		resp := httpGet(t, "/v1/stats/dynamics?program_id=1")
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
