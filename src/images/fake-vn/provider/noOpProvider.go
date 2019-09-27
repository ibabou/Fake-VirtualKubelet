package provider

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
	"time"

	"github.com/virtual-kubelet/node-cli/manager"
	"github.com/virtual-kubelet/virtual-kubelet/errdefs"
	"github.com/virtual-kubelet/virtual-kubelet/log"
	"github.com/virtual-kubelet/virtual-kubelet/node/api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	stats "k8s.io/kubernetes/pkg/kubelet/apis/stats/v1alpha1"
)

// NoOpProvider is a simulating provider.
type NoOpProvider struct {
	nodeName           string
	operatingSystem    string
	internalIP         string
	daemonEndpointPort int32
	rm                 *manager.ResourceManager
	pods               map[string]*corev1.Pod
	startTime          time.Time
	notifier           func(*corev1.Pod)
}

// NewNoOpProvider creates a new No-Op provider
func NewNoOpProvider(rm *manager.ResourceManager, nodeName, operatingSystem string, internalIP string, daemonEndpointPort int32) (*NoOpProvider, error) {
	provider := &NoOpProvider{
		nodeName:           nodeName,
		operatingSystem:    operatingSystem,
		internalIP:         internalIP,
		daemonEndpointPort: daemonEndpointPort,
		rm:                 rm,
		pods:               make(map[string]*corev1.Pod),
		startTime:          time.Now(),
		notifier:           func(*corev1.Pod) {},
	}
	return provider, nil
}

// ConfigureNode enables a provider to configure the node object that
// will be used for Kubernetes.
func (p *NoOpProvider) ConfigureNode(ctx context.Context, node *corev1.Node) {
	log.G(ctx).Infof("receive ConfigureNode")

	cap := corev1.ResourceList{
		corev1.ResourceCPU:    resource.MustParse("2000"),
		corev1.ResourceMemory: resource.MustParse("8Ti"),
		corev1.ResourcePods:   resource.MustParse("4000"),
	}
	node.Status.Capacity = cap
	node.Status.Allocatable = cap

	// All valid conditions that can be set
	node.Status.Conditions = []corev1.NodeCondition{
		{
			Type:               "Ready",
			Status:             corev1.ConditionTrue,
			LastHeartbeatTime:  metav1.Now(),
			LastTransitionTime: metav1.Now(),
			Reason:             "KubeletReady",
			Message:            "kubelet is ready.",
		},
	}

	node.Status.Addresses = []corev1.NodeAddress{
		{
			Type:    "InternalIP",
			Address: p.internalIP,
		},
	}
	node.Status.DaemonEndpoints = corev1.NodeDaemonEndpoints{
		KubeletEndpoint: corev1.DaemonEndpoint{
			Port: p.daemonEndpointPort,
		},
	}
	node.Status.NodeInfo.OperatingSystem = p.operatingSystem
}

// GetContainerLogs returns the logs of a pod by name.
func (p *NoOpProvider) GetContainerLogs(ctx context.Context, namespace, podName, containerName string, opts api.ContainerLogOpts) (io.ReadCloser, error) {
	return ioutil.NopCloser(strings.NewReader("")), nil
}

// RunInContainer executes a command in a container in the pod, copying data
// between in/out/err and the container's stdin/stdout/stderr.
func (p *NoOpProvider) RunInContainer(ctx context.Context, namespace, name, container string, cmd []string, attach api.AttachIO) error {
	return nil
}

// Implement PodLifecycleHandler

// CreatePod takes a Kubernetes Pod and deploys it within the provider.
func (p *NoOpProvider) CreatePod(ctx context.Context, pod *corev1.Pod) error {
	log.G(ctx).Infof("receive CreatePod %q", pod.Name)

	//Just add to the in-memory pod, and notify VK that everything is handled on our side
	key := p.getKey(pod)

	now := metav1.NewTime(time.Now())
	pod.Status = corev1.PodStatus{
		Phase:     corev1.PodRunning,
		HostIP:    "",
		PodIP:     "1.2.3.4", // this should be an IP that provider utilizes from within subnet
		StartTime: &now,
		Conditions: []corev1.PodCondition{
			{
				Type:   corev1.PodInitialized,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodReady,
				Status: corev1.ConditionTrue,
			},
			{
				Type:   corev1.PodScheduled,
				Status: corev1.ConditionTrue,
			},
		},
	}

	for _, container := range pod.Spec.Containers {
		pod.Status.ContainerStatuses = append(pod.Status.ContainerStatuses, corev1.ContainerStatus{
			Name:         container.Name,
			Image:        container.Image,
			Ready:        true,
			RestartCount: 0,
			State: corev1.ContainerState{
				Running: &corev1.ContainerStateRunning{
					StartedAt: now,
				},
			},
		})
	}

	p.pods[key] = pod
	p.notifier(pod)

	return nil
}

// UpdatePod takes a Kubernetes Pod and updates it within the provider.
func (p *NoOpProvider) UpdatePod(ctx context.Context, pod *corev1.Pod) error {
	log.G(ctx).Infof("receive UpdatePod %q", pod.Name)

	//This is an update from K8s side, ignoring this.
	//The main case will be when it got marked for deletion which is handled through Delete.

	return nil
}

// DeletePod takes a Kubernetes Pod and deletes it from the provider.
func (p *NoOpProvider) DeletePod(ctx context.Context, pod *corev1.Pod) error {
	log.G(ctx).Infof("receive DeletePod %q", pod.Name)

	//Just delete the in-memory pod, and notify VK with pod having the termination state and reason
	key := p.getKey(pod)

	if _, exists := p.pods[key]; !exists {
		return errdefs.NotFound("Pod not found")
	}

	delete(p.pods, key)

	now := metav1.Now()
	pod.Status.Phase = corev1.PodSucceeded
	pod.Status.Reason = "PodDeleted"

	for idx := range pod.Status.ContainerStatuses {
		pod.Status.ContainerStatuses[idx].Ready = false
		pod.Status.ContainerStatuses[idx].State = corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				Message:    "",
				FinishedAt: now,
				Reason:     "ContainerDeleted",
				StartedAt:  pod.Status.ContainerStatuses[idx].State.Running.StartedAt,
			},
		}
	}

	// Notify VK so it actually cleans up the pod from the api-server
	p.notifier(pod)

	return nil
}

// GetPod retrieves a pod by name from the provider (can be cached).
func (p *NoOpProvider) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	log.G(ctx).Infof("receive GetPod %q", name)

	//Just get the in-memory pod
	key := p.getKeyByName(namespace, name)

	if pod, exists := p.pods[key]; exists {
		return pod.DeepCopy(), nil
	}

	return nil, errdefs.NotFound("Pod not found")
}

// GetPodStatus retrieves the status of a pod by name from the provider.
func (p *NoOpProvider) GetPodStatus(ctx context.Context, namespace, name string) (*corev1.PodStatus, error) {
	log.G(ctx).Infof("receive GetPodStatus %q", name)

	//Just get the in-memory pod
	key := p.getKeyByName(namespace, name)

	if pod, exists := p.pods[key]; exists {
		return pod.Status.DeepCopy(), nil
	}

	return nil, errdefs.NotFound("Pod not found")
}

// GetPods retrieves a list of all pods running on the provider (can be cached).
func (p *NoOpProvider) GetPods(ctx context.Context) ([]*corev1.Pod, error) {
	log.G(ctx).Info("receive GetPods")

	// Create new slice to return
	var pods []*corev1.Pod
	for _, pod := range p.pods {
		pods = append(pods, pod.DeepCopy())
	}

	return pods, nil
}

// Implement optional PodNotifier (VK will pass a func for us to call when any changes happens to the pods - async updates)

// NotifyPods instructs the notifier to call the passed in function when
// the pod status changes.
// The provided pointer to a Pod is guaranteed to be used in a read-only
// fashion. The provided pod's PodStatus should be up to date when
// this function is called.
func (p *NoOpProvider) NotifyPods(ctx context.Context, notifierCb func(*corev1.Pod)) {
	// Capture the notifier to be used for communicating updates to VK
	p.notifier = notifierCb
}

// Implement optional NodeProvider

// Ping checks if the node is still active.
// This is intended to be lightweight as it will be called periodically as a
// heartbeat to keep the node marked as ready in Kubernetes.
func (p *NoOpProvider) Ping(ctx context.Context) error {
	// We're always good.
	return nil
}

// NotifyNodeStatus is used to asynchronously monitor the node.
// The passed in callback should be called any time there is a change to the
// node's status.
// This will generally trigger a call to the Kubernetes API server to update
// the status.
// The provided pointer to Node is guaranteed to be used in a read-only
// fashion.
func (p *NoOpProvider) NotifyNodeStatus(ctx context.Context, notifierCb func(*corev1.Node)) {
	// There will be no changes to the Node status from provider side for now.
}

// Implement optional PodMetricsProvider

// GetStatsSummary expose the pods/containers stats
func (p *NoOpProvider) GetStatsSummary(context.Context) (*stats.Summary, error) {
	return nil, nil
}

//

func (p *NoOpProvider) getKeyByName(namespace, name string) string {
	if namespace == "" || name == "" {
		return ""
	}

	return fmt.Sprintf("%s-%s", namespace, name)
}

func (p *NoOpProvider) getKey(pod *corev1.Pod) string {
	return p.getKeyByName(pod.ObjectMeta.Namespace, pod.ObjectMeta.Name)
}
