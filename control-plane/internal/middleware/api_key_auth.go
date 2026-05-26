package middleware

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/your-org/cortado/control-plane/internal/auth"
)

type APIKeyAuthConfig struct {
	JWKSJSON    []byte
	TenantClaim string
	Verifier    auth.FirebaseTokenVerifier
}

func NewAPIKeyAuthMiddleware(cfg APIKeyAuthConfig) func(http.Handler) http.Handler {
	jwtVerifier, err := newJWTKeyfunc(cfg.JWKSJSON)
	if err != nil {
		panic("initialize api key auth middleware: " + err.Error())
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r.Header.Get("Authorization"))
			if token != "" {
				if ctx, err := validateJWT(r.Context(), token, jwtVerifier); err == nil {
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}

				if cfg.Verifier != nil {
					if ctx, status, message, ok := validateFirebaseAPIKeyToken(
						r.Context(),
						token,
						cfg.Verifier,
						cfg.TenantClaim,
					); ok {
						next.ServeHTTP(w, r.WithContext(ctx))
						return
					} else {
						http.Error(w, message, status)
						return
					}
				}

				http.Error(w, "unauthorized", http.StatusUnauthorized)
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

func validateFirebaseAPIKeyToken(
	ctx context.Context,
	token string,
	verifier auth.FirebaseTokenVerifier,
	tenantClaim string,
) (context.Context, int, string, bool) {
	verified, err := verifier.VerifyIDToken(ctx, token)
	if err != nil {
		return nil, http.StatusUnauthorized, "unauthorized", false
	}

	nextCtx := context.WithValue(ctx, ctxKeyFirebaseToken, verified)
	nextCtx = context.WithValue(nextCtx, ctxKeyActorType, auth.ActorTypeUser)
	nextCtx = context.WithValue(nextCtx, ctxKeyUserID, strings.TrimSpace(verified.UID))

	tenantID, err := auth.TenantIDFromFirebaseClaims(verified.Claims, tenantClaim)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, auth.ErrTenantClaimMissing) {
			status = http.StatusForbidden
		}
		return nil, status, err.Error(), false
	}

	nextCtx = context.WithValue(nextCtx, ctxKeyTenantID, tenantID)
	return nextCtx, http.StatusOK, "", true
}
