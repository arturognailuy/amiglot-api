package http

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/gnailuy/amiglot-api/internal/config"
)

func TestRouter_HealthzAndLogout(t *testing.T) {
	handler := Router(config.Config{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/healthz", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.True(t, res.Code == http.StatusOK || res.Code == http.StatusNoContent)
	if res.Code == http.StatusOK {
		body := res.Body.String()
		require.Contains(t, body, "\"ok\"")
	}

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	logoutRes := httptest.NewRecorder()
	handler.ServeHTTP(logoutRes, logoutReq)
	require.True(t, logoutRes.Code == http.StatusOK || logoutRes.Code == http.StatusNoContent)
	if logoutRes.Code == http.StatusOK {
		logoutBody, _ := io.ReadAll(logoutRes.Body)
		require.True(t, strings.Contains(string(logoutBody), "\"ok\""))
	}
}

func TestRouter_ProfileUnavailable(t *testing.T) {
	handler := Router(config.Config{}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/profile", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)
	require.Equal(t, http.StatusServiceUnavailable, res.Code)
}
