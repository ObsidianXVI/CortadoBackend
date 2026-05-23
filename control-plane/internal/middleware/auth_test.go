package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDevBypassAuthRejectsMissingHeaderInDevelopment(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	handler := DevBypassAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusUnauthorized)
	}
}

func TestDevBypassAuthInjectsContextInDevelopment(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	handler := DevBypassAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tenantID, ok := TenantID(r.Context())
		if !ok || tenantID != "dev-tenant" {
			t.Fatalf("unexpected tenant context: %q %t", tenantID, ok)
		}

		userID, ok := UserID(r.Context())
		if !ok || userID != "dev-user" {
			t.Fatalf("unexpected user context: %q %t", userID, ok)
		}

		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	req.Header.Set("X-Cortado-Dev-Token", "dev-bypass")
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestDevBypassAuthAcceptsQueryTokenInDevelopment(t *testing.T) {
	t.Setenv("CORTADO_ENV", "development")

	handler := DevBypassAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test?dev_token=dev-bypass", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNoContent)
	}
}

func TestDevBypassAuthPassesThroughOutsideDevelopment(t *testing.T) {
	t.Setenv("CORTADO_ENV", "production")

	handler := DevBypassAuth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodGet, "/v1/test", nil)
	recorder := httptest.NewRecorder()

	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("unexpected status: got %d want %d", recorder.Code, http.StatusNoContent)
	}
}
