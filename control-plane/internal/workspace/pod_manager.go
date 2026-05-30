package workspace

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultAgentPort                  int32 = 9090
	defaultClusterDNSDomain                 = "cluster.local"
	defaultWorkspaceMountPath               = "/workspace"
	defaultWorkspaceNamespace               = "cortado-workspaces"
	defaultWorkspacePVCSize                 = "10Gi"
	defaultWorkspacePriorityClassName       = "workspace-priority"
	defaultWorkspaceAppName                 = "cortado-workspace-agent"
	defaultVolumeReleasePollInterval        = 250 * time.Millisecond
	defaultVolumeReleaseTimeout             = 30 * time.Second
	defaultWorkspaceStorageClass            = "cortado-workspace"
	defaultWorkspaceServiceAccount          = "workspace-sa"
	defaultQdrantImage                      = "qdrant/qdrant:v1.12.0"
	defaultQdrantMountPath                  = "/qdrant/storage"
	defaultQdrantSubPath                    = ".cortado/qdrant"
	defaultQdrantCPURequest                 = "100m"
	defaultQdrantCPULimit                   = "100m"
	defaultQdrantMemoryRequest              = "256Mi"
	defaultQdrantMemoryLimit                = "256Mi"
	workspaceAppNameLabel                   = "app.kubernetes.io/name"
	workspaceIDLabel                        = "cortado/workspace-id"
)

type StatusSink interface {
	OnPodDeleted(ctx context.Context, workspaceID string) error
	OnPodStatus(ctx context.Context, workspaceID string, phase corev1.PodPhase, ready bool, deleting bool) error
}

const (
	envGoogleCloudProject        = "GOOGLE_CLOUD_PROJECT"
	envTenantID                  = "CORTADO_TENANT_ID"
	envUsageEventsTopic          = "CORTADO_USAGE_EVENTS_TOPIC"
	envWorkspaceCPU              = "CORTADO_WORKSPACE_CPU"
	envWorkspaceID               = "CORTADO_WORKSPACE_ID"
	envWorkspaceMemoryGB         = "CORTADO_WORKSPACE_MEMORY_GB"
	envWorkspaceRegion           = "CORTADO_GCP_REGION"
	envWorkspaceSnapshotBucket   = "CORTADO_SNAPSHOT_BUCKET"
	envWorkspaceSnapshotPassword = "CORTADO_SNAPSHOT_PASSWORD"
	envWorkspaceStorageGB        = "CORTADO_WORKSPACE_STORAGE_GB"
	envWorkspaceUserID           = "CORTADO_USER_ID"
)

type PodManagerConfig struct {
	AgentPort                 int32
	ClusterPods               podListClient
	DNSDomain                 string
	Events                    eventClient
	Logf                      logfFunc
	Nodes                     nodeClient
	PVCSize                   string
	ProjectID                 string
	Region                    string
	Namespace                 string
	ServiceAccountName        string
	StorageClassName          string
	StatusSink                StatusSink
	SnapshotBucket            string
	SnapshotPassword          string
	Snapshotter               Snapshotter
	UsageEventsTopic          string
	VolumeReleasePollInterval time.Duration
	VolumeReleaseTimeout      time.Duration
}

type PodManager struct {
	agentPort                 int32
	clusterPods               podListClient
	dnsDomain                 string
	events                    eventClient
	logf                      logfFunc
	namespace                 string
	nodes                     nodeClient
	pods                      podClient
	podInformer               cache.SharedIndexInformer
	pvcs                      pvcClient
	pvcSize                   resource.Quantity
	projectID                 string
	region                    string
	runOnce                   sync.Once
	serviceAccountName        string
	services                  serviceClient
	snapshotBucket            string
	snapshotPassword          string
	snapshotter               Snapshotter
	storageClassName          string
	statusSink                StatusSink
	usageEventsTopic          string
	usageFlusher              UsageFlusher
	sleep                     func(context.Context, time.Duration) error
	volumeReleasePollInterval time.Duration
	volumeReleaseTimeout      time.Duration
}

func NewPodManager(client kubernetes.Interface, cfg PodManagerConfig) *PodManager {
	cfg = withDefaultConfig(cfg)
	if cfg.ClusterPods == nil {
		cfg.ClusterPods = client.CoreV1().Pods(metav1.NamespaceAll)
	}
	if cfg.Nodes == nil {
		cfg.Nodes = client.CoreV1().Nodes()
	}
	if cfg.Events == nil {
		cfg.Events = client.CoreV1().Events(cfg.Namespace)
	}

	return newPodManager(
		client.CoreV1().Pods(cfg.Namespace),
		client.CoreV1().PersistentVolumeClaims(cfg.Namespace),
		client.CoreV1().Services(cfg.Namespace),
		cfg,
	)
}

func newPodManager(pods podClient, pvcs pvcClient, services serviceClient, cfg PodManagerConfig) *PodManager {
	cfg = withDefaultConfig(cfg)

	podInformer := cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options metav1.ListOptions) (runtime.Object, error) {
				return pods.List(context.Background(), options)
			},
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				return pods.Watch(context.Background(), options)
			},
		},
		&corev1.Pod{},
		0,
		cache.Indexers{},
	)

	manager := &PodManager{
		agentPort:                 cfg.AgentPort,
		clusterPods:               cfg.ClusterPods,
		dnsDomain:                 cfg.DNSDomain,
		events:                    cfg.Events,
		logf:                      cfg.Logf,
		namespace:                 cfg.Namespace,
		nodes:                     cfg.Nodes,
		pods:                      pods,
		podInformer:               podInformer,
		pvcs:                      pvcs,
		pvcSize:                   resource.MustParse(cfg.PVCSize),
		projectID:                 cfg.ProjectID,
		region:                    cfg.Region,
		serviceAccountName:        cfg.ServiceAccountName,
		services:                  services,
		snapshotBucket:            cfg.SnapshotBucket,
		snapshotPassword:          cfg.SnapshotPassword,
		snapshotter:               cfg.Snapshotter,
		storageClassName:          cfg.StorageClassName,
		statusSink:                cfg.StatusSink,
		usageEventsTopic:          cfg.UsageEventsTopic,
		sleep:                     sleepWithContext,
		volumeReleasePollInterval: cfg.VolumeReleasePollInterval,
		volumeReleaseTimeout:      cfg.VolumeReleaseTimeout,
	}

	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: manager.handlePodUpsert,
		UpdateFunc: func(_, newObj interface{}) {
			manager.handlePodUpsert(newObj)
		},
		DeleteFunc: manager.handlePodDelete,
	})

	return manager
}

func withDefaultConfig(cfg PodManagerConfig) PodManagerConfig {
	if cfg.Namespace == "" {
		cfg.Namespace = defaultWorkspaceNamespace
	}
	if cfg.PVCSize == "" {
		cfg.PVCSize = defaultWorkspacePVCSize
	}
	if cfg.DNSDomain == "" {
		cfg.DNSDomain = defaultClusterDNSDomain
	}
	if cfg.ServiceAccountName == "" {
		cfg.ServiceAccountName = defaultWorkspaceServiceAccount
	}
	if cfg.StorageClassName == "" {
		cfg.StorageClassName = defaultWorkspaceStorageClass
	}
	if cfg.VolumeReleasePollInterval <= 0 {
		cfg.VolumeReleasePollInterval = defaultVolumeReleasePollInterval
	}
	if cfg.VolumeReleaseTimeout <= 0 {
		cfg.VolumeReleaseTimeout = defaultVolumeReleaseTimeout
	}
	if cfg.AgentPort == 0 {
		cfg.AgentPort = defaultAgentPort
	}
	if cfg.Logf == nil {
		cfg.Logf = log.Printf
	}
	return cfg
}

func (m *PodManager) Run(ctx context.Context) {
	m.runOnce.Do(func() {
		go m.podInformer.Run(ctx.Done())
	})
}

func (m *PodManager) SetStatusSink(statusSink StatusSink) {
	m.statusSink = statusSink
}

func (m *PodManager) SetUsageFlusher(usageFlusher UsageFlusher) {
	m.usageFlusher = usageFlusher
}

func (m *PodManager) SetSnapshotter(snapshotter Snapshotter) {
	m.snapshotter = snapshotter
}

func (m *PodManager) Create(workspace Workspace) error {
	if workspace.ID == "" {
		return errors.New("workspaceID is required")
	}
	if workspace.Image == "" {
		return errors.New("image is required")
	}

	resources, err := workspaceResources(workspace.Resources.CPU, workspace.Resources.MemoryGB)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: m.namespace,
			Labels:    workspaceLabels(workspace.ID),
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: "None",
			Ports: []corev1.ServicePort{
				{
					Name:       "grpc",
					Port:       m.agentPort,
					TargetPort: intstr.FromInt32(m.agentPort),
				},
			},
			Selector: map[string]string{
				workspaceIDLabel: workspace.ID,
			},
		},
	}

	serviceCreated := false
	if _, err := m.services.Create(context.Background(), service, metav1.CreateOptions{}); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return fmt.Errorf("create workspace service %q: %w", workspace.ID, err)
		}
	} else {
		serviceCreated = true
	}

	pvcCreated, err := m.ensurePersistentVolumeClaim(workspace.ID)
	if err != nil {
		if serviceCreated {
			if cleanupErr := m.deleteService(workspace.ID); cleanupErr != nil {
				return fmt.Errorf("create workspace pvc %q: %w (cleanup service: %v)", workspace.ID, err, cleanupErr)
			}
		}
		return fmt.Errorf("create workspace pvc %q: %w", workspace.ID, err)
	}

	if err := m.waitForVolumeRelease(workspace.ID); err != nil {
		if pvcCreated {
			if cleanupErr := m.deletePersistentVolumeClaim(workspace.ID); cleanupErr != nil {
				return fmt.Errorf("wait for workspace volume release %q: %w (cleanup pvc: %v)", workspace.ID, err, cleanupErr)
			}
		}
		if serviceCreated {
			if cleanupErr := m.deleteService(workspace.ID); cleanupErr != nil {
				return fmt.Errorf("wait for workspace volume release %q: %w (cleanup service: %v)", workspace.ID, err, cleanupErr)
			}
		}
		return fmt.Errorf("wait for workspace volume release %q: %w", workspace.ID, err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspace.ID,
			Namespace: m.namespace,
			Labels:    workspaceLabels(workspace.ID),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: m.serviceAccountName,
			PriorityClassName:  defaultWorkspacePriorityClassName,
			TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
				{
					MaxSkew:           1,
					TopologyKey:       "kubernetes.io/hostname",
					WhenUnsatisfiable: corev1.DoNotSchedule,
					LabelSelector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							workspaceAppNameLabel: defaultWorkspaceAppName,
						},
					},
				},
			},
			Containers: []corev1.Container{
				{
					Name:      "workspace",
					Image:     workspace.Image,
					Resources: resources,
					Env:       m.workspaceAgentEnv(workspace),
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: m.agentPort,
							Name:          "grpc",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace-data",
							MountPath: defaultWorkspaceMountPath,
						},
					},
				},
				{
					Name:      "qdrant",
					Image:     defaultQdrantImage,
					Resources: qdrantResources(),
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "workspace-data",
							MountPath: defaultQdrantMountPath,
							SubPath:   defaultQdrantSubPath,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "workspace-data",
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: workspacePVCName(workspace.ID),
						},
					},
				},
			},
		},
	}

	if _, err := m.pods.Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
		if pvcCreated {
			if cleanupErr := m.deletePersistentVolumeClaim(workspace.ID); cleanupErr != nil {
				return fmt.Errorf("create workspace pod %q: %w (cleanup pvc: %v)", workspace.ID, err, cleanupErr)
			}
		}
		if serviceCreated {
			if cleanupErr := m.deleteService(workspace.ID); cleanupErr != nil {
				return fmt.Errorf("create workspace pod %q: %w (cleanup service: %v)", workspace.ID, err, cleanupErr)
			}
		}
		return fmt.Errorf("create workspace pod %q: %w", workspace.ID, err)
	}

	return nil
}

func qdrantResources() corev1.ResourceRequirements {
	return corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(defaultQdrantCPURequest),
			corev1.ResourceMemory: resource.MustParse(defaultQdrantMemoryRequest),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(defaultQdrantCPULimit),
			corev1.ResourceMemory: resource.MustParse(defaultQdrantMemoryLimit),
		},
	}
}

func (m *PodManager) Stop(workspaceID string) error {
	if workspaceID == "" {
		return errors.New("workspaceID is required")
	}

	if m.snapshotter != nil {
		if err := m.snapshotter.CreateSnapshot(context.Background(), workspaceID); err != nil &&
			!isIgnorableAgentCleanupError(err) {
			return fmt.Errorf("create workspace snapshot %q: %w", workspaceID, err)
		}
	}

	return m.deletePod(workspaceID)
}

func (m *PodManager) Delete(workspaceID string) error {
	if workspaceID == "" {
		return errors.New("workspaceID is required")
	}

	if _, err := m.pods.Get(context.Background(), workspaceID, metav1.GetOptions{}); err == nil {
		if m.usageFlusher != nil {
			if err := m.usageFlusher.FlushUsageWAL(context.Background(), workspaceID); err != nil &&
				!isIgnorableAgentCleanupError(err) {
				return fmt.Errorf("flush usage WAL for workspace %q: %w", workspaceID, err)
			}
		}
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("get workspace pod %q before delete: %w", workspaceID, err)
	}

	if err := m.deletePod(workspaceID); err != nil {
		return err
	}
	if err := m.deleteService(workspaceID); err != nil {
		return err
	}
	if err := m.deletePersistentVolumeClaim(workspaceID); err != nil {
		return err
	}

	return nil
}

func (m *PodManager) GetStatus(workspaceID string) (corev1.PodPhase, error) {
	if workspaceID == "" {
		return corev1.PodUnknown, errors.New("workspaceID is required")
	}

	pod, err := m.pods.Get(context.Background(), workspaceID, metav1.GetOptions{})
	if err != nil {
		return corev1.PodUnknown, fmt.Errorf("get workspace pod %q: %w", workspaceID, err)
	}

	return pod.Status.Phase, nil
}

func (m *PodManager) GetServiceDNS(workspaceID string) string {
	pod, err := m.pods.Get(context.Background(), workspaceID, metav1.GetOptions{})
	if err == nil {
		if ip := strings.TrimSpace(pod.Status.PodIP); ip != "" {
			return ip
		}
	}
	return fmt.Sprintf("%s.%s.svc.%s", workspaceID, m.namespace, m.dnsDomain)
}

func isIgnorableAgentCleanupError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	switch status.Code(err) {
	case codes.DeadlineExceeded, codes.Canceled, codes.Unavailable, codes.Unimplemented:
		return true
	}
	message := err.Error()
	return strings.Contains(message, "context deadline exceeded") ||
		strings.Contains(message, "produced zero addresses") ||
		strings.Contains(message, "code = Unimplemented")
}

func (m *PodManager) deletePod(workspaceID string) error {
	err := m.pods.Delete(context.Background(), workspaceID, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete workspace pod %q: %w", workspaceID, err)
	}

	return nil
}

func (m *PodManager) deleteService(workspaceID string) error {
	err := m.services.Delete(context.Background(), workspaceID, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete workspace service %q: %w", workspaceID, err)
	}

	return nil
}

func (m *PodManager) handlePodDelete(obj interface{}) {
	pod, ok := podFromObject(obj)
	if !ok {
		return
	}

	workspaceID := pod.Labels[workspaceIDLabel]
	if workspaceID == "" {
		return
	}

	m.maybeLogPreemptionSitrep(context.Background(), pod)

	if m.statusSink != nil {
		_ = m.statusSink.OnPodDeleted(context.Background(), workspaceID)
	}
}

func (m *PodManager) handlePodUpsert(obj interface{}) {
	pod, ok := podFromObject(obj)
	if !ok {
		return
	}

	workspaceID := pod.Labels[workspaceIDLabel]
	if workspaceID == "" {
		return
	}

	if m.statusSink != nil {
		_ = m.statusSink.OnPodStatus(
			context.Background(),
			workspaceID,
			pod.Status.Phase,
			isPodReady(pod),
			pod.DeletionTimestamp != nil,
		)
	}
}

func (m *PodManager) maybeLogPreemptionSitrep(ctx context.Context, pod *corev1.Pod) {
	if m.events == nil || m.logf == nil || pod == nil {
		return
	}

	event, err := m.findPreemptionEvent(ctx, pod)
	if err != nil {
		m.logf("workspace preemption sitrep workspace=%s: inspect pod events: %v", pod.Name, err)
		return
	}
	if event == nil {
		return
	}

	report := m.buildPreemptionSitrep(ctx, pod, event)
	m.logf("%s", report)
}

func (m *PodManager) findPreemptionEvent(ctx context.Context, pod *corev1.Pod) (*corev1.Event, error) {
	selector := fields.AndSelectors(
		fields.OneTermEqualSelector("involvedObject.kind", "Pod"),
		fields.OneTermEqualSelector("involvedObject.name", pod.Name),
	).String()

	events, err := m.events.List(ctx, metav1.ListOptions{FieldSelector: selector})
	if err != nil {
		return nil, fmt.Errorf("list workspace pod events: %w", err)
	}

	var latest *corev1.Event
	for i := range events.Items {
		event := events.Items[i]
		if pod.UID != "" && event.InvolvedObject.UID != "" && event.InvolvedObject.UID != pod.UID {
			continue
		}
		if !isPreemptionEvent(event) {
			continue
		}
		if latest == nil || eventTimestamp(event).After(eventTimestamp(*latest)) {
			latest = event.DeepCopy()
		}
	}

	return latest, nil
}

func isPreemptionEvent(event corev1.Event) bool {
	reason := strings.ToLower(strings.TrimSpace(event.Reason))
	message := strings.ToLower(strings.TrimSpace(event.Message))
	return strings.Contains(reason, "preempt") || strings.Contains(message, "preempt")
}

func eventTimestamp(event corev1.Event) time.Time {
	switch {
	case !event.EventTime.IsZero():
		return event.EventTime.Time
	case !event.LastTimestamp.IsZero():
		return event.LastTimestamp.Time
	case !event.FirstTimestamp.IsZero():
		return event.FirstTimestamp.Time
	default:
		return event.CreationTimestamp.Time
	}
}

func (m *PodManager) buildPreemptionSitrep(ctx context.Context, pod *corev1.Pod, event *corev1.Event) string {
	var lines []string
	lines = append(lines,
		fmt.Sprintf("workspace preemption sitrep workspace=%s namespace=%s", pod.Name, pod.Namespace),
		fmt.Sprintf(
			"  pod node=%s phase=%s qos=%s ready=%t deleting=%t podIP=%s",
			emptyIfUnset(pod.Spec.NodeName),
			pod.Status.Phase,
			pod.Status.QOSClass,
			isPodReady(pod),
			pod.DeletionTimestamp != nil,
			emptyIfUnset(pod.Status.PodIP),
		),
		fmt.Sprintf(
			"  event type=%s reason=%s at=%s message=%q",
			emptyIfUnset(event.Type),
			emptyIfUnset(event.Reason),
			eventTimestamp(*event).UTC().Format(time.RFC3339),
			strings.TrimSpace(event.Message),
		),
		"  workspace containers:",
	)

	for _, container := range pod.Spec.Containers {
		lines = append(lines, fmt.Sprintf(
			"    - %s image=%s requests=%s limits=%s",
			container.Name,
			container.Image,
			formatResourceList(container.Resources.Requests),
			formatResourceList(container.Resources.Limits),
		))
	}

	if pod.Spec.NodeName != "" {
		lines = append(lines, m.describeVictimNode(ctx, pod.Spec.NodeName)...)
	}
	lines = append(lines, m.describeClusterNodes(ctx)...)

	return strings.Join(lines, "\n")
}

func (m *PodManager) describeVictimNode(ctx context.Context, nodeName string) []string {
	lines := []string{"  victim node:"}
	if m.nodes == nil {
		return append(lines, "    - unavailable: no node client configured")
	}

	node, err := m.nodes.Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return append(lines, fmt.Sprintf("    - unavailable: get node: %v", err))
	}

	podsByNode, err := m.listPodsByNode(ctx)
	if err != nil {
		lines = append(lines, fmt.Sprintf("    - node=%s allocatable=%s", node.Name, formatAllocatable(node.Status.Allocatable)))
		return append(lines, fmt.Sprintf("    - pod allocation unavailable: %v", err))
	}

	scheduled := podsByNode[nodeName]
	summary := summarizeNode(node, scheduled)
	lines = append(lines, fmt.Sprintf(
		"    - node=%s ready=%t allocatable=%s requested=%s usage=%s pods=%d",
		node.Name,
		nodeReady(node),
		formatAllocatable(node.Status.Allocatable),
		formatNodeTotals(summary.Requested),
		formatPercentages(summary.Requested, node.Status.Allocatable),
		len(scheduled),
	))

	for _, podSummary := range topPodsByRequest(scheduled, 6) {
		lines = append(lines, fmt.Sprintf(
			"      * %s/%s priority=%s qos=%s requests=%s",
			podSummary.Namespace,
			podSummary.Name,
			emptyIfUnset(podSummary.PriorityClassName),
			podSummary.QOSClass,
			formatNodeTotals(podSummary.Requested),
		))
	}

	return lines
}

func (m *PodManager) describeClusterNodes(ctx context.Context) []string {
	lines := []string{"  cluster nodes:"}
	if m.nodes == nil {
		return append(lines, "    - unavailable: no node client configured")
	}

	nodes, err := m.nodes.List(ctx, metav1.ListOptions{})
	if err != nil {
		return append(lines, fmt.Sprintf("    - unavailable: list nodes: %v", err))
	}

	podsByNode, err := m.listPodsByNode(ctx)
	if err != nil {
		lines = append(lines, fmt.Sprintf("    - pod allocation unavailable: %v", err))
	}

	sort.Slice(nodes.Items, func(i, j int) bool {
		return nodes.Items[i].Name < nodes.Items[j].Name
	})
	for i := range nodes.Items {
		node := &nodes.Items[i]
		summary := summarizeNode(node, podsByNode[node.Name])
		lines = append(lines, fmt.Sprintf(
			"    - %s ready=%t allocatable=%s requested=%s usage=%s pods=%d",
			node.Name,
			nodeReady(node),
			formatAllocatable(node.Status.Allocatable),
			formatNodeTotals(summary.Requested),
			formatPercentages(summary.Requested, node.Status.Allocatable),
			len(podsByNode[node.Name]),
		))
	}

	return lines
}

func (m *PodManager) listPodsByNode(ctx context.Context) (map[string][]*corev1.Pod, error) {
	if m.clusterPods == nil {
		return nil, errors.New("no cluster pod client configured")
	}

	podList, err := m.clusterPods.List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("list cluster pods: %w", err)
	}

	byNode := map[string][]*corev1.Pod{}
	for i := range podList.Items {
		pod := podList.Items[i].DeepCopy()
		if pod.Spec.NodeName == "" || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
			continue
		}
		byNode[pod.Spec.NodeName] = append(byNode[pod.Spec.NodeName], pod)
	}
	return byNode, nil
}

type nodeSummary struct {
	Requested resourceTotals
}

type podRequestSummary struct {
	Name              string
	Namespace         string
	PriorityClassName string
	QOSClass          corev1.PodQOSClass
	Requested         resourceTotals
}

type resourceTotals struct {
	cpuMilli       int64
	ephemeralBytes int64
	memoryBytes    int64
}

func summarizeNode(node *corev1.Node, pods []*corev1.Pod) nodeSummary {
	summary := nodeSummary{}
	for _, pod := range pods {
		summary.Requested = summary.Requested.add(effectivePodRequests(pod))
	}
	return summary
}

func topPodsByRequest(pods []*corev1.Pod, limit int) []podRequestSummary {
	summaries := make([]podRequestSummary, 0, len(pods))
	for _, pod := range pods {
		summaries = append(summaries, podRequestSummary{
			Name:              pod.Name,
			Namespace:         pod.Namespace,
			PriorityClassName: pod.Spec.PriorityClassName,
			QOSClass:          pod.Status.QOSClass,
			Requested:         effectivePodRequests(pod),
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		if summaries[i].Requested.memoryBytes != summaries[j].Requested.memoryBytes {
			return summaries[i].Requested.memoryBytes > summaries[j].Requested.memoryBytes
		}
		if summaries[i].Requested.cpuMilli != summaries[j].Requested.cpuMilli {
			return summaries[i].Requested.cpuMilli > summaries[j].Requested.cpuMilli
		}
		if summaries[i].Namespace != summaries[j].Namespace {
			return summaries[i].Namespace < summaries[j].Namespace
		}
		return summaries[i].Name < summaries[j].Name
	})

	if limit > 0 && len(summaries) > limit {
		summaries = summaries[:limit]
	}
	return summaries
}

func effectivePodRequests(pod *corev1.Pod) resourceTotals {
	var appTotals resourceTotals
	for _, container := range pod.Spec.Containers {
		appTotals = appTotals.add(resourceListTotals(container.Resources.Requests))
	}

	var maxInit resourceTotals
	for _, container := range pod.Spec.InitContainers {
		initTotals := resourceListTotals(container.Resources.Requests)
		if initTotals.cpuMilli > maxInit.cpuMilli {
			maxInit.cpuMilli = initTotals.cpuMilli
		}
		if initTotals.memoryBytes > maxInit.memoryBytes {
			maxInit.memoryBytes = initTotals.memoryBytes
		}
		if initTotals.ephemeralBytes > maxInit.ephemeralBytes {
			maxInit.ephemeralBytes = initTotals.ephemeralBytes
		}
	}

	return resourceTotals{
		cpuMilli:       max(appTotals.cpuMilli, maxInit.cpuMilli),
		memoryBytes:    max(appTotals.memoryBytes, maxInit.memoryBytes),
		ephemeralBytes: max(appTotals.ephemeralBytes, maxInit.ephemeralBytes),
	}
}

func resourceListTotals(resources corev1.ResourceList) resourceTotals {
	return resourceTotals{
		cpuMilli:       resources.Cpu().MilliValue(),
		memoryBytes:    resources.Memory().Value(),
		ephemeralBytes: resourceValue(resources, corev1.ResourceEphemeralStorage),
	}
}

func (r resourceTotals) add(other resourceTotals) resourceTotals {
	return resourceTotals{
		cpuMilli:       r.cpuMilli + other.cpuMilli,
		memoryBytes:    r.memoryBytes + other.memoryBytes,
		ephemeralBytes: r.ephemeralBytes + other.ephemeralBytes,
	}
}

func formatAllocatable(resources corev1.ResourceList) string {
	return fmt.Sprintf(
		"cpu=%s memory=%s ephemeral=%s",
		formatMilliCPU(resources.Cpu().MilliValue()),
		formatBytes(resources.Memory().Value()),
		formatBytes(resourceValue(resources, corev1.ResourceEphemeralStorage)),
	)
}

func formatNodeTotals(totals resourceTotals) string {
	return fmt.Sprintf(
		"cpu=%s memory=%s ephemeral=%s",
		formatMilliCPU(totals.cpuMilli),
		formatBytes(totals.memoryBytes),
		formatBytes(totals.ephemeralBytes),
	)
}

func formatPercentages(totals resourceTotals, allocatable corev1.ResourceList) string {
	return fmt.Sprintf(
		"cpu=%s memory=%s ephemeral=%s",
		percentString(totals.cpuMilli, allocatable.Cpu().MilliValue()),
		percentString(totals.memoryBytes, allocatable.Memory().Value()),
		percentString(totals.ephemeralBytes, resourceValue(allocatable, corev1.ResourceEphemeralStorage)),
	)
}

func percentString(used, total int64) string {
	if total <= 0 {
		return "n/a"
	}
	return fmt.Sprintf("%.0f%%", float64(used)/float64(total)*100)
}

func formatMilliCPU(milli int64) string {
	if milli%1000 == 0 {
		return fmt.Sprintf("%d", milli/1000)
	}
	return fmt.Sprintf("%.2f", float64(milli)/1000)
}

func formatBytes(bytes int64) string {
	const gib = 1024 * 1024 * 1024
	const mib = 1024 * 1024
	switch {
	case bytes == 0:
		return "0"
	case bytes%gib == 0:
		return fmt.Sprintf("%dGi", bytes/gib)
	case bytes >= gib:
		return fmt.Sprintf("%.1fGi", float64(bytes)/gib)
	case bytes%mib == 0:
		return fmt.Sprintf("%dMi", bytes/mib)
	default:
		return fmt.Sprintf("%.1fMi", float64(bytes)/mib)
	}
}

func formatResourceList(resources corev1.ResourceList) string {
	if len(resources) == 0 {
		return "none"
	}
	return fmt.Sprintf(
		"cpu=%s memory=%s ephemeral=%s",
		formatMilliCPU(resources.Cpu().MilliValue()),
		formatBytes(resources.Memory().Value()),
		formatBytes(resourceValue(resources, corev1.ResourceEphemeralStorage)),
	)
}

func emptyIfUnset(value string) string {
	if strings.TrimSpace(value) == "" {
		return "<unset>"
	}
	return value
}

func nodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func resourceValue(resources corev1.ResourceList, name corev1.ResourceName) int64 {
	quantity, ok := resources[name]
	if !ok {
		return 0
	}
	return quantity.Value()
}

func isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (m *PodManager) ensurePersistentVolumeClaim(workspaceID string) (bool, error) {
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspacePVCName(workspaceID),
			Namespace: m.namespace,
			Labels:    workspaceLabels(workspaceID),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: ptr(m.storageClassName),
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: m.pvcSize,
				},
			},
		},
	}

	if _, err := m.pvcs.Create(context.Background(), pvc, metav1.CreateOptions{}); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (m *PodManager) waitForVolumeRelease(workspaceID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), m.volumeReleaseTimeout)
	defer cancel()

	for {
		pod, err := m.pods.Get(ctx, workspaceID, metav1.GetOptions{})
		switch {
		case apierrors.IsNotFound(err):
			return nil
		case err != nil:
			return fmt.Errorf("get workspace pod %q: %w", workspaceID, err)
		case pod.DeletionTimestamp == nil:
			return nil
		}

		if err := m.sleep(ctx, m.volumeReleasePollInterval); err != nil {
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return fmt.Errorf(
					"timed out after %s waiting for terminating pod %q to release the workspace volume",
					m.volumeReleaseTimeout,
					workspaceID,
				)
			}
			return fmt.Errorf("wait for terminating pod %q: %w", workspaceID, err)
		}
	}
}

func (m *PodManager) deletePersistentVolumeClaim(workspaceID string) error {
	err := m.pvcs.Delete(context.Background(), workspacePVCName(workspaceID), metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete workspace pvc %q: %w", workspaceID, err)
	}

	return nil
}

func workspacePVCName(workspaceID string) string {
	return workspaceID + "-pvc"
}

func workspaceLabels(workspaceID string) map[string]string {
	return map[string]string{
		workspaceAppNameLabel: defaultWorkspaceAppName,
		workspaceIDLabel:      workspaceID,
	}
}

func (m *PodManager) storageGB() float64 {
	return m.pvcSize.AsApproximateFloat64() / (1024 * 1024 * 1024)
}

func (m *PodManager) workspaceAgentEnv(workspace Workspace) []corev1.EnvVar {
	env := []corev1.EnvVar{
		{Name: envGoogleCloudProject, Value: m.projectID},
		{Name: envTenantID, Value: workspace.TenantID},
		{Name: envUsageEventsTopic, Value: m.usageEventsTopic},
		{Name: envWorkspaceCPU, Value: strconv.FormatFloat(workspace.Resources.CPU, 'f', -1, 64)},
		{Name: envWorkspaceID, Value: workspace.ID},
		{Name: envWorkspaceMemoryGB, Value: strconv.FormatFloat(workspace.Resources.MemoryGB, 'f', -1, 64)},
		{Name: envWorkspaceRegion, Value: m.region},
		{Name: envWorkspaceStorageGB, Value: strconv.FormatFloat(m.storageGB(), 'f', -1, 64)},
		{Name: envWorkspaceUserID, Value: workspace.UserID},
	}
	if m.snapshotBucket != "" {
		env = append(env, corev1.EnvVar{Name: envWorkspaceSnapshotBucket, Value: m.snapshotBucket})
	}
	if m.snapshotPassword != "" {
		env = append(env, corev1.EnvVar{Name: envWorkspaceSnapshotPassword, Value: m.snapshotPassword})
	}

	return env
}

func ptr[T any](value T) *T {
	return &value
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func podFromObject(obj interface{}) (*corev1.Pod, bool) {
	if pod, ok := obj.(*corev1.Pod); ok {
		return pod, true
	}

	tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
	if !ok {
		return nil, false
	}

	pod, ok := tombstone.Obj.(*corev1.Pod)
	return pod, ok
}

func workspaceResources(cpu, memGB float64) (corev1.ResourceRequirements, error) {
	if cpu <= 0 {
		return corev1.ResourceRequirements{}, errors.New("cpu must be positive")
	}
	if memGB <= 0 {
		return corev1.ResourceRequirements{}, errors.New("memGB must be positive")
	}

	milliCPU := int64(math.Ceil(cpu * 1000))
	memoryMi := int64(math.Ceil(memGB * 1024))

	cpuQuantity := resource.MustParse(fmt.Sprintf("%dm", milliCPU))
	memoryQuantity := resource.MustParse(fmt.Sprintf("%dMi", memoryMi))

	resources := corev1.ResourceList{
		corev1.ResourceCPU:    cpuQuantity,
		corev1.ResourceMemory: memoryQuantity,
	}

	return corev1.ResourceRequirements{
		Limits:   resources,
		Requests: resources,
	}, nil
}

type podClient interface {
	Create(ctx context.Context, pod *corev1.Pod, opts metav1.CreateOptions) (*corev1.Pod, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Pod, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
}

type podListClient interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.PodList, error)
}

type pvcClient interface {
	Create(ctx context.Context, pvc *corev1.PersistentVolumeClaim, opts metav1.CreateOptions) (*corev1.PersistentVolumeClaim, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.PersistentVolumeClaim, error)
}

type serviceClient interface {
	Create(ctx context.Context, service *corev1.Service, opts metav1.CreateOptions) (*corev1.Service, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}

type eventClient interface {
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.EventList, error)
}

type nodeClient interface {
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*corev1.Node, error)
	List(ctx context.Context, opts metav1.ListOptions) (*corev1.NodeList, error)
}

type logfFunc func(format string, args ...any)
