package api

import (
	"context"
	"io"
	"net/http"

	chi "github.com/go-chi/chi/v5"
	agentpb "github.com/your-org/cortado/agent/gen/agent/v1"
	"github.com/your-org/cortado/control-plane/internal/ai"
	"github.com/your-org/cortado/control-plane/internal/auth"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	cpmiddleware "github.com/your-org/cortado/control-plane/internal/middleware"
	"github.com/your-org/cortado/control-plane/internal/tenant"
)

type RouterConfig struct {
	APIKeyAuth       func(http.Handler) http.Handler
	APIKeySvc        APIKeyService
	AICompletionSvc  AICompletionService
	ConnectHandler   http.Handler
	DevBootstrapAuth func(http.Handler) http.Handler
	DevBootstrapSvc  DevFirebaseBootstrapService
	JWKSProvider     JWKSProvider
	SessionSvc       SessionService
	TenantAuthSvc    TenantAuthProviderService
	WorkspaceFileSvc WorkspaceFileService
	WorkspaceSvc     WorkspaceService
}

type AICompletionService interface {
	StreamCompletion(ctx context.Context, params ai.CompletionParams, emit func(string) error) error
}

type APIKeyService interface {
	IssueAPIKey(ctx context.Context, tenantID, userID string) (auth.IssuedAPIKey, error)
	ListAPIKeys(ctx context.Context, tenantID, userID string) ([]auth.APIKey, error)
	RevokeAPIKey(ctx context.Context, tenantID, userID, keyID string) (auth.APIKey, error)
}

type SessionService interface {
	CreateSession(ctx context.Context, apiKey, userID string) (auth.SessionTokens, error)
	RefreshSession(ctx context.Context, refreshToken string) (string, error)
}

type TenantAuthProviderService interface {
	DeleteAuthProvider(ctx context.Context, tenantID string) error
	GetAuthProvider(ctx context.Context, tenantID string) (tenant.AuthProviderConfig, error)
	PutAuthProvider(ctx context.Context, tenantID string, input tenant.UpsertAuthProviderInput) (tenant.AuthProviderConfig, bool, error)
}

type JWKSProvider interface {
	JWKS() []byte
}

type WorkspaceFileService interface {
	DeletePath(ctx context.Context, workspaceID, path string) error
	ListDir(ctx context.Context, workspaceID, path string) ([]*agentpb.DirectoryEntry, error)
	MakeDir(ctx context.Context, workspaceID, path string) error
	ReadFile(ctx context.Context, workspaceID, path string, writer io.Writer) error
	RenamePath(ctx context.Context, workspaceID, oldPath, newPath string) error
	WriteFile(ctx context.Context, workspaceID, path string, createMissingDirs bool, reader io.Reader) (*agentpb.WriteFileResponse, error)
}

func NewRouter(cfg RouterConfig) http.Handler {
	connectHandler := cfg.ConnectHandler
	if connectHandler == nil {
		connectHandler = gateway.NewConnectHandler(gateway.ConnectHandlerConfig{})
	}
	var jwksJSON []byte
	if cfg.JWKSProvider != nil {
		jwksJSON = cfg.JWKSProvider.JWKS()
	}

	router := chi.NewRouter()

	router.Get("/health", healthHandler)
	if cfg.JWKSProvider != nil {
		router.Get("/.well-known/jwks.json", newJWKSHandler(cfg.JWKSProvider).get)
	}

	router.Route("/v1", func(r chi.Router) {
		if cfg.SessionSvc != nil {
			sessionsHandler := newSessionsHandler(cfg.SessionSvc)
			r.Post("/sessions", sessionsHandler.create)
			r.Post("/sessions/refresh", sessionsHandler.refresh)
		}
		if cfg.APIKeyAuth != nil && (cfg.APIKeySvc != nil || cfg.TenantAuthSvc != nil) {
			r.Group(func(firebaseProtected chi.Router) {
				firebaseProtected.Use(cfg.APIKeyAuth)
				if cfg.APIKeySvc != nil {
					handler := newAPIKeysHandler(cfg.APIKeySvc)
					firebaseProtected.Post("/api-keys", handler.issue)
					firebaseProtected.Get("/api-keys", handler.list)
					firebaseProtected.Delete("/api-keys/{id}", handler.revoke)
				}
				if cfg.TenantAuthSvc != nil {
					handler := newTenantAuthProviderHandler(cfg.TenantAuthSvc)
					firebaseProtected.Get("/tenant/auth-provider", handler.get)
					firebaseProtected.Put("/tenant/auth-provider", handler.put)
					firebaseProtected.Delete("/tenant/auth-provider", handler.delete)
				}
			})
		}
		if cfg.DevBootstrapSvc != nil && cfg.DevBootstrapAuth != nil {
			r.Group(func(devFirebaseProtected chi.Router) {
				devFirebaseProtected.Use(cfg.DevBootstrapAuth)
				handler := newDevBootstrapHandler(cfg.DevBootstrapSvc)
				devFirebaseProtected.Post(
					"/dev/firebase/tenant-claim",
					handler.assignTenantClaim,
				)
			})
		}
		r.Group(func(protected chi.Router) {
			protected.Use(cpmiddleware.NewAuthMiddleware(cpmiddleware.AuthConfig{JWKSJSON: jwksJSON}))
			if cfg.WorkspaceSvc != nil {
				handler := newWorkspacesHandler(cfg.WorkspaceSvc)
				protected.Get("/workspaces", handler.list)
				protected.Post("/workspaces", handler.create)
				protected.Get("/workspaces/{id}", handler.get)
				protected.Post("/workspaces/{id}/start", handler.start)
				protected.Post("/workspaces/{id}/stop", handler.stop)
				protected.Delete("/workspaces/{id}", handler.delete)
				if cfg.WorkspaceFileSvc != nil {
					filesHandler := newFilesHandler(cfg.WorkspaceSvc, cfg.WorkspaceFileSvc)
					protected.Get("/workspaces/{id}/files", filesHandler.list)
					protected.Delete("/workspaces/{id}/files", filesHandler.delete)
					protected.Post("/workspaces/{id}/files/directory", filesHandler.makeDir)
					protected.Post("/workspaces/{id}/files/rename", filesHandler.rename)
					protected.Get("/workspaces/{id}/files/content", filesHandler.readContent)
					protected.Put("/workspaces/{id}/files/content", filesHandler.writeContent)
				}
				if cfg.AICompletionSvc != nil {
					aiHandler := newAIHandler(cfg.WorkspaceSvc, cfg.AICompletionSvc)
					protected.Post("/workspaces/{id}/ai/complete", aiHandler.complete)
				}
			}
			protected.Method(http.MethodGet, "/workspaces/{id}/connect", connectHandler)
		})
	})

	return router
}
