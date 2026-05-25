package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/your-org/cortado/control-plane/internal/auth"
)

type contextKey string

const (
	ctxKeyTenantID      contextKey = "tenant_id"
	ctxKeyUserID        contextKey = "user_id"
	ctxKeyFirebaseToken contextKey = "firebase_token"
)

type AuthConfig struct {
	JWKSJSON []byte
}

func TenantID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(ctxKeyTenantID).(string)
	return value, ok
}

func UserID(ctx context.Context) (string, bool) {
	value, ok := ctx.Value(ctxKeyUserID).(string)
	return value, ok
}

func FirebaseToken(ctx context.Context) (*auth.VerifiedFirebaseToken, bool) {
	value, ok := ctx.Value(ctxKeyFirebaseToken).(*auth.VerifiedFirebaseToken)
	return value, ok && value != nil
}

func NewAuthMiddleware(cfg AuthConfig) func(http.Handler) http.Handler {
	verifier, err := newJWTKeyfunc(cfg.JWKSJSON)
	if err != nil {
		panic(fmt.Sprintf("initialize auth middleware: %v", err))
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token := requestToken(r); token != "" {
				ctx, err := validateJWT(r.Context(), token, verifier)
				if err != nil {
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				}
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			if os.Getenv("CORTADO_ENV") == "development" && isDevBypassRequest(r) {
				next.ServeHTTP(w, injectDevContext(r))
				return
			}

			http.Error(w, "unauthorized", http.StatusUnauthorized)
		})
	}
}

func newJWTKeyfunc(jwksJSON []byte) (keyfunc.Keyfunc, error) {
	if len(strings.TrimSpace(string(jwksJSON))) == 0 {
		return nil, nil
	}
	return keyfunc.NewJWKSetJSON(json.RawMessage(jwksJSON))
}

func requestToken(r *http.Request) string {
	if token := bearerToken(r.Header.Get("Authorization")); token != "" {
		return token
	}
	if websocket.IsWebSocketUpgrade(r) {
		return strings.TrimSpace(r.URL.Query().Get("token"))
	}
	return ""
}

func bearerToken(headerValue string) string {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return ""
	}

	scheme, token, ok := strings.Cut(headerValue, " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return ""
	}
	return strings.TrimSpace(token)
}

func validateJWT(ctx context.Context, tokenString string, verifier keyfunc.Keyfunc) (context.Context, error) {
	if verifier == nil {
		return nil, errors.New("jwt verifier is not configured")
	}

	claims := &auth.AccessClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		verifier.Keyfunc,
		jwt.WithValidMethods([]string{"RS256"}),
	)
	if err != nil || !token.Valid {
		return nil, errors.New("invalid jwt")
	}
	if strings.TrimSpace(claims.TenantID) == "" || strings.TrimSpace(claims.Subject) == "" || claims.ExpiresAt == nil {
		return nil, errors.New("required claims are missing")
	}

	ctx = context.WithValue(ctx, ctxKeyTenantID, claims.TenantID)
	ctx = context.WithValue(ctx, ctxKeyUserID, claims.Subject)
	return ctx, nil
}

func injectDevContext(r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), ctxKeyTenantID, "dev-tenant")
	ctx = context.WithValue(ctx, ctxKeyUserID, "dev-user")
	return r.WithContext(ctx)
}

func isDevBypassRequest(r *http.Request) bool {
	token := strings.TrimSpace(r.Header.Get("X-Cortado-Dev-Token"))
	if token == "" && websocket.IsWebSocketUpgrade(r) {
		token = strings.TrimSpace(r.URL.Query().Get("dev_token"))
	}
	return token == "dev-bypass"
}
