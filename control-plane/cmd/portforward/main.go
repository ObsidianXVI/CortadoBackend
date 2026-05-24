package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/auth"
	"github.com/your-org/cortado/control-plane/internal/gateway"
	"github.com/your-org/cortado/control-plane/internal/portforward"
	"github.com/your-org/cortado/control-plane/internal/store"
	"github.com/your-org/cortado/control-plane/internal/workspace"
)

const defaultHTTPPort = "8080"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	projectID := gcpProjectID()
	if projectID == "" {
		log.Fatal("missing GCP project id")
	}

	firestoreClient, err := newFirestoreClient(ctx, projectID)
	if err != nil {
		log.Fatalf("initialize firestore client: %v", err)
	}
	defer func() {
		if closeErr := firestoreClient.Close(); closeErr != nil {
			log.Printf("close firestore client: %v", closeErr)
		}
	}()

	authService, err := newSessionService(firestoreClient)
	if err != nil {
		log.Fatalf("initialize auth service: %v", err)
	}
	defer func() {
		if closeErr := authService.Close(); closeErr != nil {
			log.Printf("close auth service: %v", closeErr)
		}
	}()

	workspaceRepository := store.NewFirestoreWorkspaceStore(firestoreClient, store.FirestoreWorkspaceStoreConfig{
		Collection: envOrDefault("CORTADO_FIRESTORE_COLLECTION", "workspaces"),
	})
	workspaceService := workspace.NewService(workspace.ServiceConfig{
		Repository: workspaceRepository,
	})

	resolver := gateway.StaticWorkspaceResolver{
		Namespace: os.Getenv("CORTADO_WORKSPACE_NAMESPACE"),
		DNSDomain: os.Getenv("CORTADO_CLUSTER_DNS_DOMAIN"),
	}
	portService := workspace.NewAgentPortService(workspace.AgentPortServiceConfig{
		WorkspaceResolver: resolver,
	})

	handler := portforward.NewHandler(portforward.HandlerConfig{
		PortService:       portService,
		WorkspaceResolver: resolver,
		WorkspaceService:  workspaceService,
	})

	server := &http.Server{
		Addr:              ":" + port,
		Handler:           portforward.NewRouter(portforward.RouterConfig{Handler: handler, JWKSProvider: authService}),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown http server: %v", err)
		}
	}()

	log.Printf("portforward listening on :%s", port)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func gcpProjectID() string {
	for _, value := range []string{
		os.Getenv("GCP_PROJECT"),
		os.Getenv("GOOGLE_CLOUD_PROJECT"),
		os.Getenv("GCLOUD_PROJECT"),
	} {
		if value != "" {
			return value
		}
	}
	return ""
}

func newFirestoreClient(ctx context.Context, projectID string) (*firestore.Client, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create firestore client: %w", err)
	}
	return client, nil
}

func newSessionService(firestoreClient *firestore.Client) (*auth.Service, error) {
	authStore := store.NewFirestoreAuthStore(firestoreClient, store.FirestoreAuthStoreConfig{
		APIKeysCollection:       os.Getenv("CORTADO_AUTH_API_KEYS_COLLECTION"),
		RefreshTokensCollection: os.Getenv("CORTADO_AUTH_REFRESH_TOKENS_COLLECTION"),
	})

	service, err := auth.NewService(auth.ServiceConfig{
		Cache:         auth.NewValidationCacheFromEnv(),
		PrivateKeyPEM: os.Getenv("CORTADO_JWT_PRIVATE_KEY_PEM"),
		Repository:    authStore,
	})
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	return service, nil
}
