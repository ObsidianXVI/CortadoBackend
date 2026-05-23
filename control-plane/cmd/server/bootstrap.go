package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/your-org/cortado/control-plane/internal/store"
	"github.com/your-org/cortado/control-plane/internal/workspace"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	container "google.golang.org/api/container/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const cloudPlatformScope = "https://www.googleapis.com/auth/cloud-platform"

func newWorkspaceService(ctx context.Context) (*workspace.Service, error) {
	projectID := gcpProjectID()
	if projectID == "" {
		return nil, errors.New("missing GCP project id")
	}

	firestoreClient, err := firestore.NewClient(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("create firestore client: %w", err)
	}
	go func() {
		<-ctx.Done()
		_ = firestoreClient.Close()
	}()

	kubeClient, err := newKubernetesClient(ctx, projectID)
	if err != nil {
		_ = firestoreClient.Close()
		return nil, fmt.Errorf("create kubernetes client: %w", err)
	}

	repository := store.NewFirestoreWorkspaceStore(firestoreClient, store.FirestoreWorkspaceStoreConfig{
		Collection: os.Getenv("CORTADO_FIRESTORE_COLLECTION"),
	})
	podManager := workspace.NewPodManager(kubeClient, workspace.PodManagerConfig{
		DNSDomain:        os.Getenv("CORTADO_CLUSTER_DNS_DOMAIN"),
		Namespace:        os.Getenv("CORTADO_WORKSPACE_NAMESPACE"),
		PVCSize:          os.Getenv("CORTADO_WORKSPACE_PVC_SIZE"),
		StorageClassName: os.Getenv("CORTADO_WORKSPACE_STORAGE_CLASS"),
	})
	service := workspace.NewService(workspace.ServiceConfig{
		Provisioner: podManager,
		Repository:  repository,
	})
	podManager.SetStatusSink(service)
	podManager.Run(ctx)

	return service, nil
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
