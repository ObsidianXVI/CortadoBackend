package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/ai"
	"github.com/your-org/cortado/control-plane/internal/auth"
	"github.com/your-org/cortado/control-plane/internal/store"
	"github.com/your-org/cortado/control-plane/internal/tenant"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	container "google.golang.org/api/container/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

func newFirestoreClient(ctx context.Context, projectID string) (*firestore.Client, error) {
	client, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create firestore client: %w", err)
	}
	return client, nil
}

func newWorkspaceService(ctx context.Context, projectID string, firestoreClient *firestore.Client) (*workspace.Service, *workspace.PodManager, error) {
	kubeClient, err := newKubernetesClient(ctx, projectID)
	if err != nil {
		return nil, nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	repository := store.NewFirestoreWorkspaceStore(firestoreClient, store.FirestoreWorkspaceStoreConfig{
		Collection: os.Getenv("CORTADO_FIRESTORE_COLLECTION"),
	})
	podManager := workspace.NewPodManager(kubeClient, workspace.PodManagerConfig{
		DNSDomain:        os.Getenv("CORTADO_CLUSTER_DNS_DOMAIN"),
		Namespace:        os.Getenv("CORTADO_WORKSPACE_NAMESPACE"),
		PVCSize:          os.Getenv("CORTADO_WORKSPACE_PVC_SIZE"),
		ProjectID:        projectID,
		Region:           os.Getenv("CORTADO_GKE_CLUSTER_LOCATION"),
		SnapshotBucket:   os.Getenv("CORTADO_SNAPSHOT_BUCKET"),
		SnapshotPassword: os.Getenv("CORTADO_SNAPSHOT_PASSWORD"),
		StorageClassName: os.Getenv("CORTADO_WORKSPACE_STORAGE_CLASS"),
		UsageEventsTopic: os.Getenv("CORTADO_USAGE_EVENTS_TOPIC"),
	})
	service := workspace.NewService(workspace.ServiceConfig{
		Provisioner: podManager,
		Repository:  repository,
	})
	podManager.SetStatusSink(service)
	podManager.SetUsageFlusher(workspace.NewAgentUsageFlusher(workspace.AgentUsageFlusherConfig{
		WorkspaceResolver: podManager,
	}))
	podManager.SetSnapshotter(workspace.NewAgentSnapshotter(workspace.AgentSnapshotterConfig{
		WorkspaceResolver: podManager,
	}))
	podManager.Run(ctx)

	return service, podManager, nil
}

func newAuthStore(firestoreClient *firestore.Client) *store.FirestoreAuthStore {
	return store.NewFirestoreAuthStore(firestoreClient, store.FirestoreAuthStoreConfig{
		APIKeysCollection:         os.Getenv("CORTADO_AUTH_API_KEYS_COLLECTION"),
		FirstPartyUsersCollection: os.Getenv("CORTADO_AUTH_FIRST_PARTY_USERS_COLLECTION"),
		RefreshTokensCollection:   os.Getenv("CORTADO_AUTH_REFRESH_TOKENS_COLLECTION"),
		TenantsCollection:         os.Getenv("CORTADO_TENANTS_COLLECTION"),
	})
}

func newTenantStore(firestoreClient *firestore.Client) *store.FirestoreTenantStore {
	return store.NewFirestoreTenantStore(firestoreClient, store.FirestoreTenantStoreConfig{
		Collection: os.Getenv("CORTADO_TENANTS_COLLECTION"),
	})
}

func newSessionService(repository auth.Repository, firebaseVerifier auth.FirebaseTokenVerifier) (*auth.Service, error) {
	service, err := auth.NewService(auth.ServiceConfig{
		Cache:            auth.NewValidationCacheFromEnv(),
		FirebaseVerifier: firebaseVerifier,
		PrivateKeyPEM:    os.Getenv("CORTADO_JWT_PRIVATE_KEY_PEM"),
		Repository:       repository,
	})
	if err != nil {
		return nil, fmt.Errorf("create auth service: %w", err)
	}
	return service, nil
}

func newTenantAuthProviderService(repository tenant.Repository) (*tenant.Service, error) {
	service, err := tenant.NewService(tenant.ServiceConfig{
		Repository: repository,
	})
	if err != nil {
		return nil, fmt.Errorf("create tenant auth provider service: %w", err)
	}
	return service, nil
}

func newAPIKeyService(repository auth.APIKeyRepository) (*auth.APIKeyService, error) {
	service, err := auth.NewAPIKeyService(auth.APIKeyServiceConfig{
		Repository: repository,
	})
	if err != nil {
		return nil, fmt.Errorf("create api key service: %w", err)
	}
	return service, nil
}

func newPlatformTenantService(
	repository auth.PlatformTenantRepository,
	apiKeyService *auth.APIKeyService,
) (*auth.PlatformTenantService, error) {
	service, err := auth.NewPlatformTenantService(auth.PlatformTenantServiceConfig{
		APIKeys:    apiKeyService,
		Repository: repository,
	})
	if err != nil {
		return nil, fmt.Errorf("create platform tenant service: %w", err)
	}
	return service, nil
}

func newFirebaseVerifier(ctx context.Context, projectID string) (*auth.FirebaseVerifier, error) {
	firebaseProjectID := envOrDefault("CORTADO_FIREBASE_PROJECT_ID", projectID)
	verifier, err := auth.NewFirebaseVerifier(ctx, firebaseProjectID)
	if err != nil {
		return nil, fmt.Errorf("create firebase verifier: %w", err)
	}
	return verifier, nil
}

func newDevFirebaseBootstrapService(
	manager auth.FirebaseClaimsManager,
) (*auth.DevFirebaseBootstrapService, error) {
	service, err := auth.NewDevFirebaseBootstrapService(
		auth.DevFirebaseBootstrapConfig{
			DefaultTenantID: envOrDefault(
				"CORTADO_FIREBASE_DEV_TENANT_ID",
				"demo-tenant",
			),
			Enabled:     os.Getenv("CORTADO_ENV") == "development",
			Manager:     manager,
			TenantClaim: envOrDefault("CORTADO_FIREBASE_TENANT_CLAIM", "tenant_id"),
		},
	)
	if err != nil {
		return nil, fmt.Errorf("create dev firebase bootstrap service: %w", err)
	}
	return service, nil
}

func newAIService(projectID string, resolver ai.WorkspaceResolver) (*ai.Service, error) {
	embedder, err := ai.NewVertexEmbedder(ai.VertexEmbedderConfig{
		Dimensions: envInt("CORTADO_VERTEX_DIMENSIONS", aiDefaultVertexDimensions()),
		Location:   envOrDefault("CORTADO_VERTEX_LOCATION", "us-central1"),
		Model:      envOrDefault("CORTADO_VERTEX_MODEL", "text-embedding-004"),
		ProjectID:  envOrDefault("CORTADO_VERTEX_PROJECT_ID", projectID),
		TaskType:   envOrDefault("CORTADO_VERTEX_QUERY_TASK_TYPE", "RETRIEVAL_QUERY"),
	})
	if err != nil {
		return nil, fmt.Errorf("create vertex embedder: %w", err)
	}

	provider := ai.NewGeminiProvider(ai.GeminiProviderConfig{
		APIKey:          os.Getenv("CORTADO_AI_API_KEY"),
		MaxOutputTokens: envInt("CORTADO_AI_MAX_OUTPUT_TOKENS", 256),
		Model:           envOrDefault("CORTADO_AI_MODEL", "gemini-2.5-flash"),
		Temperature:     envFloat("CORTADO_AI_TEMPERATURE", 0.2),
	})

	return ai.NewService(ai.ServiceConfig{
		MaxPrefixBytes: envInt("CORTADO_AI_PREFIX_BYTES", 4*1024),
		MaxSuffixBytes: envInt("CORTADO_AI_SUFFIX_BYTES", 1*1024),
		MaxResults:     envInt("CORTADO_AI_RETRIEVAL_LIMIT", 3),
		Provider:       provider,
		QueryEmbedder:  embedder,
		QueryLines:     envInt("CORTADO_AI_QUERY_LINE_COUNT", 5),
		Retriever: ai.NewQdrantClient(ai.QdrantClientConfig{
			Port:              envInt("CORTADO_QDRANT_PORT", 6333),
			WorkspaceResolver: resolver,
		}),
	}), nil
}

func newKubernetesClient(ctx context.Context, projectID string) (kubernetes.Interface, error) {
	config, err := newKubernetesConfig(ctx, projectID)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("build kubernetes client: %w", err)
	}
	return client, nil
}

func newKubernetesConfig(ctx context.Context, projectID string) (*rest.Config, error) {
	for _, path := range []string{
		os.Getenv("CORTADO_KUBECONFIG"),
		os.Getenv("KUBECONFIG"),
	} {
		if path == "" {
			continue
		}
		config, err := clientcmd.BuildConfigFromFlags("", path)
		if err != nil {
			return nil, fmt.Errorf("load kubeconfig %q: %w", path, err)
		}
		return config, nil
	}

	homeDir, err := os.UserHomeDir()
	if err == nil {
		defaultPath := filepath.Join(homeDir, ".kube", "config")
		if _, statErr := os.Stat(defaultPath); statErr == nil {
			config, loadErr := clientcmd.BuildConfigFromFlags("", defaultPath)
			if loadErr != nil {
				return nil, fmt.Errorf("load kubeconfig %q: %w", defaultPath, loadErr)
			}
			return config, nil
		}
	}

	if config, err := rest.InClusterConfig(); err == nil {
		return config, nil
	}

	clusterName := os.Getenv("CORTADO_GKE_CLUSTER_NAME")
	clusterLocation := os.Getenv("CORTADO_GKE_CLUSTER_LOCATION")
	if clusterName != "" && clusterLocation != "" && projectID != "" {
		return gkeConfigFromCluster(ctx, projectID, clusterLocation, clusterName)
	}

	host := os.Getenv("CORTADO_KUBE_HOST")
	if host == "" {
		return nil, errors.New("missing kubernetes configuration")
	}

	caData, err := kubeCAData()
	if err != nil {
		return nil, err
	}

	tokenSource, err := google.DefaultTokenSource(ctx, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes token source: %w", err)
	}

	config := &rest.Config{
		Host: strings.TrimSpace(host),
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauthRoundTripper{
				base:        rt,
				tokenSource: tokenSource,
			}
		},
	}

	return config, nil
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

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envFloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func aiDefaultVertexDimensions() int {
	return 768
}

func gkeConfigFromCluster(ctx context.Context, projectID, location, clusterName string) (*rest.Config, error) {
	service, err := container.NewService(ctx)
	if err != nil {
		return nil, fmt.Errorf("create container service: %w", err)
	}

	clusterPath := fmt.Sprintf("projects/%s/locations/%s/clusters/%s", projectID, location, clusterName)
	cluster, err := service.Projects.Locations.Clusters.Get(clusterPath).Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("load cluster %q: %w", clusterPath, err)
	}

	caData, err := base64.StdEncoding.DecodeString(cluster.MasterAuth.ClusterCaCertificate)
	if err != nil {
		return nil, fmt.Errorf("decode cluster CA certificate: %w", err)
	}

	tokenSource, err := google.DefaultTokenSource(ctx, cloudPlatformScope)
	if err != nil {
		return nil, fmt.Errorf("create kubernetes token source: %w", err)
	}

	return &rest.Config{
		Host: "https://" + strings.TrimSpace(cluster.Endpoint),
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caData,
		},
		WrapTransport: func(rt http.RoundTripper) http.RoundTripper {
			return &oauthRoundTripper{
				base:        rt,
				tokenSource: tokenSource,
			}
		},
	}, nil
}

func kubeCAData() ([]byte, error) {
	if pem := os.Getenv("CORTADO_KUBE_CA_CERT"); strings.TrimSpace(pem) != "" {
		return []byte(pem), nil
	}

	encoded := os.Getenv("CORTADO_KUBE_CA_CERT_BASE64")
	if encoded == "" {
		return nil, errors.New("missing kubernetes CA certificate")
	}

	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, fmt.Errorf("decode kubernetes CA certificate: %w", err)
	}
	return data, nil
}

type oauthRoundTripper struct {
	base        http.RoundTripper
	tokenSource oauth2.TokenSource
}

func (t *oauthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	token, err := t.tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("fetch access token: %w", err)
	}

	clone := req.Clone(req.Context())
	clone.Header.Set("Authorization", "Bearer "+token.AccessToken)
	return t.base.RoundTrip(clone)
}
