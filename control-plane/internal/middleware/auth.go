package middleware

import (
	"context"
	"net/http"
	"os"
)

type contextKey string

const (
	ctxKeyTenantID contextKey = "tenant_id"
	ctxKeyUserID   contextKey = "user_id"
)

func TenantID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(ctxKeyTenantID).(string)
	return value, ok
}

func UserID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(ctxKeyUserID).(string)
	return value, ok
}

func DevBypassAuth(next http.Handler) http.Handler {
	if os.Getenv("CORTADO_ENV") != "development" {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Cortado-Dev-Token")
		if token == "" {
			token = r.URL.Query().Get("dev_token")
		}
		if token != "dev-bypass" {
			http.Error(w, "missing dev bypass token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ctxKeyTenantID, "dev-tenant")
		ctx = context.WithValue(ctx, ctxKeyUserID, "dev-user")
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
