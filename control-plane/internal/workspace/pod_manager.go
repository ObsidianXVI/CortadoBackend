package workspace

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

const (
	defaultAgentPort               int32 = 9090
	defaultWorkspaceNamespace            = "cortado-workspaces"
	defaultWorkspaceServiceAccount       = "workspace-sa"
	workspaceIDLabel                     = "cortado/workspace-id"
)

type StatusSink interface {
	DeleteWorkspace(ctx context.Context, workspaceID string) error
	SetWorkspacePhase(ctx context.Context, workspaceID string, phase corev1.PodPhase) error
}

type PodManagerConfig struct {
	AgentPort          int32
	Namespace          string
	ServiceAccountName string
	StatusSink         StatusSink
}

type PodManager struct {
	agentPort          int32
	namespace          string
	pods               podClient
	podInformer        cache.SharedIndexInformer
	runOnce            sync.Once
	serviceAccountName string
	services           serviceClient
	statusSink         StatusSink
}

func NewPodManager(client kubernetes.Interface, cfg PodManagerConfig) *PodManager {
	cfg = withDefaultConfig(cfg)

	return newPodManager(
		client.CoreV1().Pods(cfg.Namespace),
		client.CoreV1().Services(cfg.Namespace),
		cfg,
	)
}

func newPodManager(pods podClient, services serviceClient, cfg PodManagerConfig) *PodManager {
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
		agentPort:          cfg.AgentPort,
		namespace:          cfg.Namespace,
		pods:               pods,
		podInformer:        podInformer,
		serviceAccountName: cfg.ServiceAccountName,
		services:           services,
		statusSink:         cfg.StatusSink,
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
	if cfg.ServiceAccountName == "" {
		cfg.ServiceAccountName = defaultWorkspaceServiceAccount
	}
	if cfg.AgentPort == 0 {
		cfg.AgentPort = defaultAgentPort
	}
	return cfg
}

func (m *PodManager) Run(ctx context.Context) {
	m.runOnce.Do(func() {
		go m.podInformer.Run(ctx.Done())
	})
}

func (m *PodManager) Create(workspaceID, image string, cpu, memGB float64) error {
	if workspaceID == "" {
		return errors.New("workspaceID is required")
	}
	if image == "" {
		return errors.New("image is required")
	}

	resources, err := workspaceResources(cpu, memGB)
	if err != nil {
		return err
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspaceID,
			Namespace: m.namespace,
			Labels: map[string]string{
				workspaceIDLabel: workspaceID,
			},
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
				workspaceIDLabel: workspaceID,
			},
		},
	}

	if _, err := m.services.Create(context.Background(), service, metav1.CreateOptions{}); err != nil {
		return fmt.Errorf("create workspace service %q: %w", workspaceID, err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      workspaceID,
			Namespace: m.namespace,
			Labels: map[string]string{
				workspaceIDLabel: workspaceID,
			},
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: m.serviceAccountName,
			Containers: []corev1.Container{
				{
					Name:      "workspace",
					Image:     image,
					Resources: resources,
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: m.agentPort,
							Name:          "grpc",
						},
					},
				},
			},
		},
	}

	if _, err := m.pods.Create(context.Background(), pod, metav1.CreateOptions{}); err != nil {
		if cleanupErr := m.deleteService(workspaceID); cleanupErr != nil {
			return fmt.Errorf("create workspace pod %q: %w (cleanup service: %v)", workspaceID, err, cleanupErr)
		}
		return fmt.Errorf("create workspace pod %q: %w", workspaceID, err)
	}

	return nil
}

func (m *PodManager) Delete(workspaceID string) error {
	if workspaceID == "" {
		return errors.New("workspaceID is required")
	}

	if err := m.deletePod(workspaceID); err != nil {
		return err
	}
	if err := m.deleteService(workspaceID); err != nil {
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
	return fmt.Sprintf("%s.%s.svc.cluster.local", workspaceID, m.namespace)
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
	if m.statusSink == nil {
		return
	}

	pod, ok := podFromObject(obj)
	if !ok {
		return
	}

	workspaceID := pod.Labels[workspaceIDLabel]
	if workspaceID == "" {
		return
	}

	_ = m.statusSink.DeleteWorkspace(context.Background(), workspaceID)
}

func (m *PodManager) handlePodUpsert(obj interface{}) {
	if m.statusSink == nil {
		return
	}

	pod, ok := podFromObject(obj)
	if !ok {
		return
	}

	workspaceID := pod.Labels[workspaceIDLabel]
	if workspaceID == "" {
		return
	}

	_ = m.statusSink.SetWorkspacePhase(context.Background(), workspaceID, pod.Status.Phase)
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

type serviceClient interface {
	Create(ctx context.Context, service *corev1.Service, opts metav1.CreateOptions) (*corev1.Service, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
}
