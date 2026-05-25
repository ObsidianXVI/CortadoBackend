package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/your-org/cortado/control-plane/internal/auth"
)

type FirebaseAuthConfig struct {
	TenantClaim string
	Verifier    auth.FirebaseTokenVerifier
}

func NewFirebaseAuthMiddleware(cfg FirebaseAuthConfig) func(http.Handler) http.Handler {
	if cfg.Verifier == nil {
		panic("initialize firebase auth middleware: verifier is required")
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			verified, err := cfg.Verifier.VerifyIDToken(r.Context(), token)
			if err != nil {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}

			tenantID, err := auth.TenantIDFromFirebaseClaims(verified.Claims, cfg.TenantClaim)
			if err != nil {
				status := http.StatusUnauthorized
				if errors.Is(err, auth.ErrTenantClaimMissing) {
					status = http.StatusForbidden
				}
				http.Error(w, err.Error(), status)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKeyTenantID, tenantID)
			ctx = context.WithValue(ctx, ctxKeyUserID, strings.TrimSpace(verified.UID))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
