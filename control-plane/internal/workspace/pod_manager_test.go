package workspace

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

func TestPodManagerCreateCreatesHeadlessServiceAndPod(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{})

	if err := manager.Create(Workspace{
		ID:       "ws-123",
		TenantID: "tenant-1",
		UserID:   "user-1",
		Image:    "example.com/cortado/workspace:test",
		Resources: Resources{
			CPU:      0.5,
			MemoryGB: 2,
		},
	}); err != nil {
		t.Fatalf("create workspace pod: %v", err)
	}

	service, err := services.Get("ws-123")
	if err != nil {
		t.Fatalf("get workspace service: %v", err)
	}
	if service.Spec.ClusterIP != "None" {
		t.Fatalf("unexpected service clusterIP: %q", service.Spec.ClusterIP)
	}
	if service.Spec.Selector[workspaceIDLabel] != "ws-123" {
		t.Fatalf("unexpected service selector: %#v", service.Spec.Selector)
	}

	pod, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get workspace pod: %v", err)
	}
	if pod.Spec.ServiceAccountName != defaultWorkspaceServiceAccount {
		t.Fatalf("unexpected pod service account: %q", pod.Spec.ServiceAccountName)
	}
	if pod.Spec.PriorityClassName != defaultWorkspacePriorityClassName {
		t.Fatalf("unexpected pod priority class: %q", pod.Spec.PriorityClassName)
	}
	if pod.Labels[workspaceIDLabel] != "ws-123" {
		t.Fatalf("unexpected pod labels: %#v", pod.Labels)
	}
	if got := pod.Spec.Containers[0].Resources.Requests.Cpu().MilliValue(); got != 500 {
		t.Fatalf("unexpected cpu request: got %d want %d", got, 500)
	}
	if got := pod.Spec.Containers[0].Resources.Requests.Memory().Value(); got != 2*1024*1024*1024 {
		t.Fatalf("unexpected memory request: got %d want %d", got, 2*1024*1024*1024)
	}
	if len(pod.Spec.Containers) != 2 {
		t.Fatalf("unexpected container count: got %d want %d", len(pod.Spec.Containers), 2)
	}
	qdrant := pod.Spec.Containers[1]
	if qdrant.Name != "qdrant" {
		t.Fatalf("unexpected qdrant container name: %q", qdrant.Name)
	}
	if qdrant.Image != defaultQdrantImage {
		t.Fatalf("unexpected qdrant image: %q", qdrant.Image)
	}
	if got := qdrant.Resources.Requests.Cpu().String(); got != defaultQdrantCPURequest {
		t.Fatalf("unexpected qdrant cpu request: got %q want %q", got, defaultQdrantCPURequest)
	}
	if got := qdrant.Resources.Requests.Memory().String(); got != defaultQdrantMemoryRequest {
		t.Fatalf("unexpected qdrant memory request: got %q want %q", got, defaultQdrantMemoryRequest)
	}
	if got := qdrant.Resources.Limits.Cpu().String(); got != defaultQdrantCPULimit {
		t.Fatalf("unexpected qdrant cpu limit: got %q want %q", got, defaultQdrantCPULimit)
	}
	if got := qdrant.Resources.Limits.Memory().String(); got != defaultQdrantMemoryLimit {
		t.Fatalf("unexpected qdrant memory limit: got %q want %q", got, defaultQdrantMemoryLimit)
	}
	if len(qdrant.VolumeMounts) != 1 {
		t.Fatalf("unexpected qdrant volume mounts: %#v", qdrant.VolumeMounts)
	}
	if qdrant.VolumeMounts[0].MountPath != defaultQdrantMountPath || qdrant.VolumeMounts[0].SubPath != defaultQdrantSubPath {
		t.Fatalf("unexpected qdrant volume mount: %#v", qdrant.VolumeMounts[0])
	}
	if pod.Spec.Volumes[0].PersistentVolumeClaim == nil || pod.Spec.Volumes[0].PersistentVolumeClaim.ClaimName != "ws-123-pvc" {
		t.Fatalf("unexpected pod pvc volume: %#v", pod.Spec.Volumes[0])
	}
	pvc, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get workspace pvc: %v", err)
	}
	if pvc.Labels[workspaceIDLabel] != "ws-123" {
		t.Fatalf("unexpected pvc labels: %#v", pvc.Labels)
	}
	if pvc.Spec.StorageClassName == nil || *pvc.Spec.StorageClassName != defaultWorkspaceStorageClass {
		t.Fatalf("unexpected pvc storage class: %#v", pvc.Spec.StorageClassName)
	}
	if len(pvc.Spec.AccessModes) != 1 || pvc.Spec.AccessModes[0] != corev1.ReadWriteOnce {
		t.Fatalf("unexpected pvc access modes: %#v", pvc.Spec.AccessModes)
	}
	if got := pvc.Spec.Resources.Requests.Storage().String(); got != defaultWorkspacePVCSize {
		t.Fatalf("unexpected pvc size: got %q want %q", got, defaultWorkspacePVCSize)
	}
}

func TestPodManagerCreateInjectsUsageEnv(t *testing.T) {
	pods := newMemoryPodClient()
	manager := newPodManager(
		pods,
		newMemoryPVCClient(),
		newMemoryServiceClient(),
		PodManagerConfig{
			PVCSize:          "10Gi",
			ProjectID:        "cortado-ide",
			Region:           "us-central1",
			SnapshotBucket:   "cortado-snapshots-cortado-ide-dev",
			SnapshotPassword: "snapshot-secret",
			UsageEventsTopic: "cortado-usage-events-dev",
		},
	)

	if err := manager.Create(Workspace{
		ID:       "ws-123",
		TenantID: "tenant-1",
		UserID:   "user-1",
		Image:    "example.com/cortado/workspace:test",
		Resources: Resources{
			CPU:      1,
			MemoryGB: 2,
		},
	}); err != nil {
		t.Fatalf("create workspace pod: %v", err)
	}

	pod, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("get workspace pod: %v", err)
	}

	env := map[string]string{}
	for _, entry := range pod.Spec.Containers[0].Env {
		env[entry.Name] = entry.Value
	}
	if env[envGoogleCloudProject] != "cortado-ide" {
		t.Fatalf("unexpected project env: %#v", env)
	}
	if env[envUsageEventsTopic] != "cortado-usage-events-dev" {
		t.Fatalf("unexpected topic env: %#v", env)
	}
	if env[envWorkspaceSnapshotBucket] != "cortado-snapshots-cortado-ide-dev" {
		t.Fatalf("unexpected snapshot bucket env: %#v", env)
	}
	if env[envWorkspaceSnapshotPassword] != "snapshot-secret" {
		t.Fatalf("unexpected snapshot password env: %#v", env)
	}
	if env[envWorkspaceID] != "ws-123" || env[envTenantID] != "tenant-1" || env[envWorkspaceUserID] != "user-1" {
		t.Fatalf("unexpected workspace identity env: %#v", env)
	}
}

func TestPodManagerCreateWaitsForTerminatingPodToReleaseVolume(t *testing.T) {
	deletingAt := metav1.NewTime(time.Date(2026, time.May, 23, 20, 0, 0, 0, time.UTC))
	pods := &sequencedPodClient{
		getResponses: []podGetResponse{
			{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ws-123",
						Namespace:         defaultWorkspaceNamespace,
						DeletionTimestamp: &deletingAt,
					},
				},
			},
			{
				err: apierrors.NewNotFound(corev1.Resource("pods"), "ws-123"),
			},
		},
	}
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{
		VolumeReleasePollInterval: time.Millisecond,
		VolumeReleaseTimeout:      10 * time.Millisecond,
	})
	manager.sleep = func(_ context.Context, _ time.Duration) error { return nil }

	if err := manager.Create(Workspace{
		ID:       "ws-123",
		TenantID: "tenant-1",
		UserID:   "user-1",
		Image:    "example.com/cortado/workspace:test",
		Resources: Resources{
			CPU:      1,
			MemoryGB: 2,
		},
	}); err != nil {
		t.Fatalf("create workspace pod after wait: %v", err)
	}

	if pods.getCalls < 2 {
		t.Fatalf("expected at least 2 pod lookups, got %d", pods.getCalls)
	}
	if pods.createCalls != 1 {
		t.Fatalf("expected exactly 1 pod create call, got %d", pods.createCalls)
	}
	if _, err := services.Get("ws-123"); err != nil {
		t.Fatalf("get workspace service after create: %v", err)
	}
	if _, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{}); err != nil {
		t.Fatalf("get workspace pvc after create: %v", err)
	}
}

func TestPodManagerCreateTimesOutWhilePodIsStillTerminating(t *testing.T) {
	deletingAt := metav1.NewTime(time.Date(2026, time.May, 23, 20, 5, 0, 0, time.UTC))
	pods := &sequencedPodClient{
		getResponses: []podGetResponse{
			{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ws-123",
						Namespace:         defaultWorkspaceNamespace,
						DeletionTimestamp: &deletingAt,
					},
				},
				repeat: true,
			},
		},
	}
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{
		VolumeReleasePollInterval: time.Millisecond,
		VolumeReleaseTimeout:      3 * time.Millisecond,
	})

	err := manager.Create(Workspace{
		ID:       "ws-123",
		TenantID: "tenant-1",
		UserID:   "user-1",
		Image:    "example.com/cortado/workspace:test",
		Resources: Resources{
			CPU:      1,
			MemoryGB: 2,
		},
	})
	if err == nil {
		t.Fatal("expected create to time out while pod is still terminating")
	}
	if got := err.Error(); got == "" || !containsAll(got, "wait for workspace volume release", "timed out") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
	if pods.createCalls != 0 {
		t.Fatalf("expected no pod create call, got %d", pods.createCalls)
	}
	if _, err := services.Get("ws-123"); !apierrors.IsNotFound(err) {
		t.Fatalf("expected service cleanup after timeout, got %v", err)
	}
	if _, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pvc cleanup after timeout, got %v", err)
	}
}

func TestPodManagerDeleteRemovesPodAndService(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = services.Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = pvcs.Create(context.Background(), &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123-pvc",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{})
	flusher := &usageFlusherStub{}
	manager.SetUsageFlusher(flusher)
	snapshotter := &snapshotterStub{}
	manager.SetSnapshotter(snapshotter)

	if err := manager.Delete("ws-123"); err != nil {
		t.Fatalf("delete workspace resources: %v", err)
	}
	if flusher.workspaceID != "ws-123" {
		t.Fatalf("unexpected flushed workspace: %q", flusher.workspaceID)
	}
	if snapshotter.workspaceID != "" {
		t.Fatalf("delete should not create a snapshot, got workspace %q", snapshotter.workspaceID)
	}

	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod to be deleted, got %v", err)
	}
	if _, err := services.Get("ws-123"); !apierrors.IsNotFound(err) {
		t.Fatalf("expected service to be deleted, got %v", err)
	}
	if _, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pvc to be deleted, got %v", err)
	}
}

func TestPodManagerStopCreatesSnapshotBeforeDeletingPod(t *testing.T) {
	pods := newMemoryPodClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})
	snapshotter := &snapshotterStub{}
	manager.SetSnapshotter(snapshotter)

	if err := manager.Stop("ws-123"); err != nil {
		t.Fatalf("stop workspace resources: %v", err)
	}
	if snapshotter.workspaceID != "ws-123" {
		t.Fatalf("unexpected snapshot workspace: %q", snapshotter.workspaceID)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod to be deleted after stop, got %v", err)
	}
}

func TestPodManagerStopIgnoresSnapshotTimeout(t *testing.T) {
	pods := newMemoryPodClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})
	manager.SetSnapshotter(&snapshotterStub{err: context.DeadlineExceeded})

	if err := manager.Stop("ws-123"); err != nil {
		t.Fatalf("stop workspace with snapshot timeout: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod deletion after snapshot timeout, got %v", err)
	}
}

func TestPodManagerStopIgnoresSnapshotDeadlineExceededStatus(t *testing.T) {
	pods := newMemoryPodClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})
	manager.SetSnapshotter(&snapshotterStub{
		err: status.Error(codes.DeadlineExceeded, "context deadline exceeded while waiting for connections to become ready"),
	})

	if err := manager.Stop("ws-123"); err != nil {
		t.Fatalf("stop workspace with deadline-exceeded snapshot agent: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod deletion after ignored snapshot failure, got %v", err)
	}
}

func TestPodManagerStopIgnoresUnavailableSnapshotAgent(t *testing.T) {
	pods := newMemoryPodClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})
	manager.SetSnapshotter(&snapshotterStub{
		err: errors.New("rpc error: code = Unavailable desc = name resolver error: produced zero addresses"),
	})

	if err := manager.Stop("ws-123"); err != nil {
		t.Fatalf("stop workspace with unavailable snapshot agent: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod deletion after ignored snapshot failure, got %v", err)
	}
}

func TestPodManagerStopIgnoresUnimplementedSnapshotAgent(t *testing.T) {
	pods := newMemoryPodClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})
	manager.SetSnapshotter(&snapshotterStub{
		err: errors.New("rpc error: code = Unimplemented desc = unknown method CreateSnapshot for service agent.v1.WorkspaceAgentService"),
	})

	if err := manager.Stop("ws-123"); err != nil {
		t.Fatalf("stop workspace with unimplemented snapshot agent: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod deletion after ignored snapshot failure, got %v", err)
	}
}

func TestPodManagerGetStatusReturnsPodPhase(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{})

	phase, err := manager.GetStatus("ws-123")
	if err != nil {
		t.Fatalf("get workspace status: %v", err)
	}
	if phase != corev1.PodRunning {
		t.Fatalf("unexpected workspace phase: got %q want %q", phase, corev1.PodRunning)
	}
}

func TestPodManagerDeleteIgnoresUnavailableUsageFlusher(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = services.Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = pvcs.Create(context.Background(), &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123-pvc",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{})
	manager.SetUsageFlusher(&usageFlusherStub{
		err: errors.New("rpc error: code = Unavailable desc = name resolver error: produced zero addresses"),
	})

	if err := manager.Delete("ws-123"); err != nil {
		t.Fatalf("delete workspace with unavailable usage flusher: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod to be deleted, got %v", err)
	}
	if _, err := services.Get("ws-123"); !apierrors.IsNotFound(err) {
		t.Fatalf("expected service to be deleted, got %v", err)
	}
	if _, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pvc to be deleted, got %v", err)
	}
}

func TestPodManagerDeleteIgnoresDeadlineExceededUsageFlusher(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = services.Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = pvcs.Create(context.Background(), &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123-pvc",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{})
	manager.SetUsageFlusher(&usageFlusherStub{
		err: status.Error(codes.DeadlineExceeded, "context deadline exceeded while waiting for connections to become ready"),
	})

	if err := manager.Delete("ws-123"); err != nil {
		t.Fatalf("delete workspace with deadline-exceeded usage flusher: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod to be deleted, got %v", err)
	}
	if _, err := services.Get("ws-123"); !apierrors.IsNotFound(err) {
		t.Fatalf("expected service to be deleted, got %v", err)
	}
	if _, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pvc to be deleted, got %v", err)
	}
}

func TestPodManagerDeleteIgnoresUnimplementedUsageFlusher(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = services.Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	_, _ = pvcs.Create(context.Background(), &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123-pvc",
			Namespace: defaultWorkspaceNamespace,
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{})
	manager.SetUsageFlusher(&usageFlusherStub{
		err: errors.New("rpc error: code = Unimplemented desc = unknown method FlushUsageWAL for service agent.v1.WorkspaceAgentService"),
	})

	if err := manager.Delete("ws-123"); err != nil {
		t.Fatalf("delete workspace with unimplemented usage flusher: %v", err)
	}
	if _, err := pods.Get(context.Background(), "ws-123", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pod to be deleted, got %v", err)
	}
	if _, err := services.Get("ws-123"); !apierrors.IsNotFound(err) {
		t.Fatalf("expected service to be deleted, got %v", err)
	}
	if _, err := pvcs.Get(context.Background(), "ws-123-pvc", metav1.GetOptions{}); !apierrors.IsNotFound(err) {
		t.Fatalf("expected pvc to be deleted, got %v", err)
	}
}

func TestPodManagerGetServiceDNS(t *testing.T) {
	manager := newPodManager(newMemoryPodClient(), newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})

	if got := manager.GetServiceDNS("ws-123"); got != "ws-123.cortado-workspaces.svc.cluster.local" {
		t.Fatalf("unexpected service dns: %q", got)
	}
}

func TestPodManagerGetServiceDNSUsesPodIPWhenAvailable(t *testing.T) {
	pods := newMemoryPodClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
		},
		Status: corev1.PodStatus{
			PodIP: "10.13.129.18",
		},
	}, metav1.CreateOptions{})
	manager := newPodManager(pods, newMemoryPVCClient(), newMemoryServiceClient(), PodManagerConfig{})

	if got := manager.GetServiceDNS("ws-123"); got != "10.13.129.18" {
		t.Fatalf("unexpected workspace target: %q", got)
	}
}

func TestPodManagerGetServiceDNSUsesConfiguredDomain(t *testing.T) {
	manager := newPodManager(
		newMemoryPodClient(),
		newMemoryPVCClient(),
		newMemoryServiceClient(),
		PodManagerConfig{DNSDomain: "cortado-dev.internal"},
	)

	if got := manager.GetServiceDNS("ws-123"); got != "ws-123.cortado-workspaces.svc.cortado-dev.internal" {
		t.Fatalf("unexpected service dns: %q", got)
	}
}

func TestPodManagerRunPublishesPodLifecycleEvents(t *testing.T) {
	pods := newMemoryPodClient()
	pvcs := newMemoryPVCClient()
	services := newMemoryServiceClient()
	_, _ = pods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-123",
			Namespace: defaultWorkspaceNamespace,
			Labels: map[string]string{
				workspaceIDLabel: "ws-123",
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodPending,
		},
	}, metav1.CreateOptions{})
	sink := &statusSinkStub{
		deleteCh: make(chan string, 1),
		phaseCh:  make(chan phaseEvent, 1),
	}
	manager := newPodManager(pods, pvcs, services, PodManagerConfig{StatusSink: sink})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	manager.Run(ctx)

	select {
	case event := <-sink.phaseCh:
		if event.workspaceID != "ws-123" || event.phase != corev1.PodPending || event.ready || event.deleting {
			t.Fatalf("unexpected phase event: %#v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for phase event")
	}

	if err := pods.Delete(context.Background(), "ws-123", metav1.DeleteOptions{}); err != nil {
		t.Fatalf("delete workspace pod: %v", err)
	}

	select {
	case workspaceID := <-sink.deleteCh:
		if workspaceID != "ws-123" {
			t.Fatalf("unexpected deleted workspace id: %q", workspaceID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for delete event")
	}
}

func TestPodManagerHandlePodDeleteLogsPreemptionSitrep(t *testing.T) {
	clusterPods := newMemoryPodClient()
	logger := &logRecorder{}
	manager := newPodManager(
		newMemoryPodClient(),
		newMemoryPVCClient(),
		newMemoryServiceClient(),
		PodManagerConfig{
			ClusterPods: clusterPods,
			Events: newMemoryEventClient([]corev1.Event{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "ws-123.18c8f7e3fcb6ec32",
						Namespace:         defaultWorkspaceNamespace,
						CreationTimestamp: metav1.NewTime(time.Date(2026, time.May, 30, 10, 0, 0, 0, time.UTC)),
					},
					InvolvedObject: corev1.ObjectReference{
						Kind:      "Pod",
						Name:      "ws-123",
						Namespace: defaultWorkspaceNamespace,
						UID:       "pod-uid",
					},
					Type:    corev1.EventTypeWarning,
					Reason:  "Preempted",
					Message: "Preempted by pod kube-system/gke-system-balloon-pod-dhrjs on node-a",
				},
			}),
			Logf:  logger.Logf,
			Nodes: newMemoryNodeClient([]corev1.Node{testNode("node-a", "16", "64Gi")}),
		},
	)

	workspacePod := testWorkspacePod("ws-123", "pod-uid", "node-a", 8, 16)
	_, _ = clusterPods.Create(context.Background(), workspacePod, metav1.CreateOptions{})
	_, _ = clusterPods.Create(context.Background(), &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "gke-system-balloon-pod-dhrjs",
			Namespace: "kube-system",
		},
		Spec: corev1.PodSpec{
			NodeName:          "node-a",
			PriorityClassName: "system-node-critical",
			Containers: []corev1.Container{
				{
					Name: "balloon",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("12"),
							corev1.ResourceMemory: resource.MustParse("56Gi"),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{QOSClass: corev1.PodQOSGuaranteed},
	}, metav1.CreateOptions{})

	manager.handlePodDelete(cache.DeletedFinalStateUnknown{Obj: workspacePod.DeepCopy()})

	if len(logger.lines) != 1 {
		t.Fatalf("expected exactly one sitrep log, got %d", len(logger.lines))
	}

	report := logger.lines[0]
	if !containsAll(
		report,
		"workspace preemption sitrep workspace=ws-123 namespace=cortado-workspaces",
		"reason=Preempted",
		"node=node-a",
		"kube-system/gke-system-balloon-pod-dhrjs",
		"cluster nodes:",
	) {
		t.Fatalf("unexpected sitrep report:\n%s", report)
	}
}

func TestPodManagerHandlePodDeleteSkipsSitrepWithoutPreemptionEvent(t *testing.T) {
	logger := &logRecorder{}
	manager := newPodManager(
		newMemoryPodClient(),
		newMemoryPVCClient(),
		newMemoryServiceClient(),
		PodManagerConfig{
			ClusterPods: newMemoryPodClient(),
			Events:      newMemoryEventClient(nil),
			Logf:        logger.Logf,
			Nodes:       newMemoryNodeClient(nil),
		},
	)

	manager.handlePodDelete(testWorkspacePod("ws-123", "pod-uid", "node-a", 1, 2))

	if len(logger.lines) != 0 {
		t.Fatalf("expected no sitrep logs, got %d", len(logger.lines))
	}
}

type phaseEvent struct {
	deleting    bool
	phase       corev1.PodPhase
	ready       bool
	workspaceID string
}

type statusSinkStub struct {
	deleteCh chan string
	phaseCh  chan phaseEvent
}

type usageFlusherStub struct {
	err         error
	workspaceID string
}

type snapshotterStub struct {
	err         error
	workspaceID string
}

type logRecorder struct {
	lines []string
}

func (s *statusSinkStub) OnPodDeleted(_ context.Context, workspaceID string) error {
	s.deleteCh <- workspaceID
	return nil
}

func (s *statusSinkStub) OnPodStatus(_ context.Context, workspaceID string, phase corev1.PodPhase, ready bool, deleting bool) error {
	s.phaseCh <- phaseEvent{
		deleting:    deleting,
		phase:       phase,
		ready:       ready,
		workspaceID: workspaceID,
	}
	return nil
}

func (u *usageFlusherStub) FlushUsageWAL(_ context.Context, workspaceID string) error {
	u.workspaceID = workspaceID
	return u.err
}

func (s *snapshotterStub) CreateSnapshot(_ context.Context, workspaceID string) error {
	s.workspaceID = workspaceID
	return s.err
}

func (l *logRecorder) Logf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

type memoryPodClient struct {
	mu      sync.Mutex
	objects map[string]*corev1.Pod
	watcher *watch.RaceFreeFakeWatcher
}

func newMemoryPodClient() *memoryPodClient {
	return &memoryPodClient{
		objects: map[string]*corev1.Pod{},
		watcher: watch.NewRaceFreeFake(),
	}
}

func (c *memoryPodClient) Create(_ context.Context, pod *corev1.Pod, _ metav1.CreateOptions) (*corev1.Pod, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.objects[pod.Name]; exists {
		return nil, apierrors.NewAlreadyExists(corev1.Resource("pods"), pod.Name)
	}

	copy := pod.DeepCopy()
	c.objects[pod.Name] = copy
	c.watcher.Add(copy.DeepCopy())
	return copy.DeepCopy(), nil
}

func (c *memoryPodClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	pod, exists := c.objects[name]
	if !exists {
		return apierrors.NewNotFound(corev1.Resource("pods"), name)
	}

	delete(c.objects, name)
	c.watcher.Delete(pod.DeepCopy())
	return nil
}

func (c *memoryPodClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Pod, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pod, exists := c.objects[name]
	if !exists {
		return nil, apierrors.NewNotFound(corev1.Resource("pods"), name)
	}

	return pod.DeepCopy(), nil
}

func (c *memoryPodClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.PodList, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	list := &corev1.PodList{}
	for _, pod := range c.objects {
		list.Items = append(list.Items, *pod.DeepCopy())
	}

	return list, nil
}

func (c *memoryPodClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return c.watcher, nil
}

type memoryServiceClient struct {
	mu      sync.Mutex
	objects map[string]*corev1.Service
}

func newMemoryServiceClient() *memoryServiceClient {
	return &memoryServiceClient{
		objects: map[string]*corev1.Service{},
	}
}

func (c *memoryServiceClient) Create(_ context.Context, service *corev1.Service, _ metav1.CreateOptions) (*corev1.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.objects[service.Name]; exists {
		return nil, apierrors.NewAlreadyExists(corev1.Resource("services"), service.Name)
	}

	copy := service.DeepCopy()
	c.objects[service.Name] = copy
	return copy.DeepCopy(), nil
}

func (c *memoryServiceClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.objects[name]; !exists {
		return apierrors.NewNotFound(corev1.Resource("services"), name)
	}

	delete(c.objects, name)
	return nil
}

func (c *memoryServiceClient) Get(name string) (*corev1.Service, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	service, exists := c.objects[name]
	if !exists {
		return nil, apierrors.NewNotFound(corev1.Resource("services"), name)
	}

	return service.DeepCopy(), nil
}

func (c *memoryServiceClient) String() string {
	return fmt.Sprintf("memoryServiceClient{%d}", len(c.objects))
}

type memoryEventClient struct {
	events []corev1.Event
}

func newMemoryEventClient(events []corev1.Event) *memoryEventClient {
	return &memoryEventClient{events: events}
}

func (c *memoryEventClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.EventList, error) {
	list := &corev1.EventList{}
	for _, event := range c.events {
		list.Items = append(list.Items, *event.DeepCopy())
	}
	return list, nil
}

type memoryNodeClient struct {
	nodes map[string]*corev1.Node
}

func newMemoryNodeClient(nodes []corev1.Node) *memoryNodeClient {
	objects := make(map[string]*corev1.Node, len(nodes))
	for i := range nodes {
		node := nodes[i].DeepCopy()
		objects[node.Name] = node
	}
	return &memoryNodeClient{nodes: objects}
}

func (c *memoryNodeClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Node, error) {
	node, ok := c.nodes[name]
	if !ok {
		return nil, apierrors.NewNotFound(corev1.Resource("nodes"), name)
	}
	return node.DeepCopy(), nil
}

func (c *memoryNodeClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.NodeList, error) {
	list := &corev1.NodeList{}
	for _, node := range c.nodes {
		list.Items = append(list.Items, *node.DeepCopy())
	}
	return list, nil
}

type memoryPVCClient struct {
	mu      sync.Mutex
	objects map[string]*corev1.PersistentVolumeClaim
}

func newMemoryPVCClient() *memoryPVCClient {
	return &memoryPVCClient{
		objects: map[string]*corev1.PersistentVolumeClaim{},
	}
}

func (c *memoryPVCClient) Create(_ context.Context, pvc *corev1.PersistentVolumeClaim, _ metav1.CreateOptions) (*corev1.PersistentVolumeClaim, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.objects[pvc.Name]; exists {
		return nil, apierrors.NewAlreadyExists(corev1.Resource("persistentvolumeclaims"), pvc.Name)
	}

	copy := pvc.DeepCopy()
	c.objects[pvc.Name] = copy
	return copy.DeepCopy(), nil
}

func (c *memoryPVCClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.objects[name]; !exists {
		return apierrors.NewNotFound(corev1.Resource("persistentvolumeclaims"), name)
	}

	delete(c.objects, name)
	return nil
}

func (c *memoryPVCClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.PersistentVolumeClaim, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	pvc, exists := c.objects[name]
	if !exists {
		return nil, apierrors.NewNotFound(corev1.Resource("persistentvolumeclaims"), name)
	}

	return pvc.DeepCopy(), nil
}

type sequencedPodClient struct {
	mu           sync.Mutex
	createCalls  int
	createdPod   *corev1.Pod
	getCalls     int
	getResponses []podGetResponse
}

type podGetResponse struct {
	err    error
	pod    *corev1.Pod
	repeat bool
}

func (c *sequencedPodClient) Create(_ context.Context, pod *corev1.Pod, _ metav1.CreateOptions) (*corev1.Pod, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.createCalls++
	c.createdPod = pod.DeepCopy()
	return c.createdPod.DeepCopy(), nil
}

func (c *sequencedPodClient) Delete(_ context.Context, name string, _ metav1.DeleteOptions) error {
	return apierrors.NewNotFound(corev1.Resource("pods"), name)
}

func (c *sequencedPodClient) Get(_ context.Context, name string, _ metav1.GetOptions) (*corev1.Pod, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.getCalls++
	if len(c.getResponses) == 0 {
		return nil, apierrors.NewNotFound(corev1.Resource("pods"), name)
	}

	response := c.getResponses[0]
	if !response.repeat {
		c.getResponses = c.getResponses[1:]
	}
	if response.err != nil {
		return nil, response.err
	}
	return response.pod.DeepCopy(), nil
}

func (c *sequencedPodClient) List(_ context.Context, _ metav1.ListOptions) (*corev1.PodList, error) {
	return &corev1.PodList{}, nil
}

func (c *sequencedPodClient) Watch(_ context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return watch.NewRaceFreeFake(), nil
}

func containsAll(haystack string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(haystack, needle) {
			return false
		}
	}
	return true
}

func testWorkspacePod(name, uid, nodeName string, cpu, memoryGi float64) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: defaultWorkspaceNamespace,
			UID:       types.UID(uid),
			Labels: map[string]string{
				workspaceIDLabel: name,
			},
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{
				{
					Name:  "workspace",
					Image: "example.com/workspace:latest",
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%gm", cpu*1000)),
							corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%gGi", memoryGi)),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse(fmt.Sprintf("%gm", cpu*1000)),
							corev1.ResourceMemory: resource.MustParse(fmt.Sprintf("%gGi", memoryGi)),
						},
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase:    corev1.PodRunning,
			PodIP:    "10.13.0.10",
			QOSClass: corev1.PodQOSBurstable,
			Conditions: []corev1.PodCondition{{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
}

func testNode(name, cpu, memory string) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpu),
				corev1.ResourceMemory: resource.MustParse(memory),
			},
			Conditions: []corev1.NodeCondition{{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionTrue,
			}},
		},
	}
}
