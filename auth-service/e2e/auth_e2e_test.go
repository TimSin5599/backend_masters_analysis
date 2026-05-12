//go:build e2e

// E2E тесты для auth-service с Testcontainers.
// Запуск: go test -tags=e2e ./e2e/... -v -timeout 120s
// Требуют запущенный Docker.
//
// Пользователь для тестов: admin@masters.com / password
// (засевается автоматически из init_schema.sql)
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// httpPost выполняет POST-запрос. Если cookieName/cookieValue непустые — выставляет Cookie-заголовок напрямую,
// минуя sanitizeCookieValue (который может сломать JWT в Go 1.23+).
func httpPost(t *testing.T, path string, body interface{}, cookieName, cookieValue string) *http.Response {
	t.Helper()
	b, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, srvURL+path, bytes.NewBuffer(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	if cookieName != "" && cookieValue != "" {
		req.Header.Set("Cookie", fmt.Sprintf("%s=%s", cookieName, cookieValue))
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func httpGet(t *testing.T, path, token string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, srvURL+path, nil)
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// refreshTokenValue извлекает значение refresh_token cookie из Set-Cookie заголовков ответа.
func refreshTokenValue(resp *http.Response) string {
	for _, c := range resp.Cookies() {
		if c.Name == "refresh_token" {
			return c.Value
		}
	}
	return ""
}

// seedCredentials — данные засеянного admin-пользователя из init_schema.sql.
const (
	seedEmail    = "admin@masters.com"
	seedPassword = "password"
)

// TestE2E_AuthFlow тестирует полный цикл: логин → /me → обновление токена → логаут → отказ в refresh.
func TestE2E_AuthFlow(t *testing.T) {
	// 1. Логин
	var accessToken string
	var rtValue string // raw значение refresh-токена из Set-Cookie
	t.Run("Login", func(t *testing.T) {
		resp := httpPost(t, "/v1/login", map[string]string{
			"email": seedEmail, "password": seedPassword,
		}, "", "")
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		accessToken = body["access_token"]
		assert.NotEmpty(t, accessToken)

		rtValue = refreshTokenValue(resp)
		assert.NotEmpty(t, rtValue, "refresh_token cookie should be set")
	})

	// 2. /me — защищённый эндпоинт
	t.Run("GetCurrentUser", func(t *testing.T) {
		resp := httpGet(t, "/v1/auth/me", accessToken)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&body)
		assert.Equal(t, seedEmail, body["email"])
	})

	// 3. Обновление токена (ротация)
	var newRTValue string
	t.Run("RefreshToken", func(t *testing.T) {
		if rtValue == "" {
			t.Skip("no refresh token")
		}
		resp := httpPost(t, "/v1/refresh", nil, "refresh_token", rtValue)
		defer resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode)

		var body map[string]string
		json.NewDecoder(resp.Body).Decode(&body)
		newAccessToken := body["access_token"]
		assert.NotEmpty(t, newAccessToken)

		// Захватываем новый refresh-токен (старый инвалидирован ротацией)
		newRTValue = refreshTokenValue(resp)
		assert.NotEmpty(t, newRTValue, "new refresh_token cookie should be set after rotation")
	})

	// 4. Логаут (используем новый токен, если Refresh прошёл; иначе — оригинальный)
	logoutRT := newRTValue
	if logoutRT == "" {
		logoutRT = rtValue
	}
	t.Run("Logout", func(t *testing.T) {
		resp := httpPost(t, "/v1/logout", nil, "refresh_token", logoutRT)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	// 5. После логаута старый refresh-токен не должен работать
	t.Run("RefreshAfterLogout", func(t *testing.T) {
		if logoutRT == "" {
			t.Skip("no refresh token")
		}
		resp := httpPost(t, "/v1/refresh", nil, "refresh_token", logoutRT)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})
}

// TestE2E_Login_InvalidCredentials проверяет отказ при неверных данных.
func TestE2E_Login_InvalidCredentials(t *testing.T) {
	resp := httpPost(t, "/v1/login", map[string]string{
		"email": "nonexistent@test.com", "password": "wrong",
	}, "", "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestE2E_ProtectedRoute_NoToken проверяет 401 без токена.
func TestE2E_ProtectedRoute_NoToken(t *testing.T) {
	resp := httpGet(t, "/v1/auth/me", "")
	defer resp.Body.Close()
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestE2E_CreateUser_AsAdmin проверяет создание пользователя через защищённый POST /v1/users.
func TestE2E_CreateUser_AsAdmin(t *testing.T) {
	loginResp := httpPost(t, "/v1/login", map[string]string{
		"email": seedEmail, "password": seedPassword,
	}, "", "")
	defer loginResp.Body.Close()
	require.Equal(t, http.StatusOK, loginResp.StatusCode)

	var loginBody map[string]string
	json.NewDecoder(loginResp.Body).Decode(&loginBody)
	token := loginBody["access_token"]
	require.NotEmpty(t, token)

	req, err := http.NewRequest(http.MethodPost, srvURL+"/v1/users",
		bytes.NewBufferString(`{"email":"e2e_newuser@test.com","password":"pass1234","roles":["manager"]}`),
	)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 201 Created или 409 Conflict (если тест запускался повторно)
	assert.True(t, resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusConflict,
		"unexpected status %d", resp.StatusCode)
}
