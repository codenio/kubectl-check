package plugin

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// podRestartAttentionThreshold is the minimum RestartCount on any container (or init container)
// that flags a pod for attention even when phase is Running and all containers are Ready.
const podRestartAttentionThreshold int32 = 5

// AuditPods returns pods that need attention as a PodList. totalInScope is len(Items) from the
// unfiltered list; benignInScope counts running/ready pods without high restart counts.
func AuditPods(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*corev1.PodList, int, int, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create clientset: %w", err)
	}

	namespace, err := namespaceForQuery(configFlags, o.AllNamespaces)
	if err != nil {
		return nil, 0, 0, err
	}

	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list pods: %w", err)
	}

	totalInScope := len(pods.Items)
	benignInScope := 0
	filtered := make([]corev1.Pod, 0, totalInScope)
	for i := range pods.Items {
		if podIsBenign(pods.Items[i]) {
			benignInScope++
		} else {
			filtered = append(filtered, pods.Items[i])
		}
	}
	return &corev1.PodList{Items: filtered}, totalInScope, benignInScope, nil
}

// podIsHealthy reports Running phase with all containers ready (desired steady state).
func podIsHealthy(pod corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}
	for _, cs := range pod.Status.ContainerStatuses {
		if !cs.Ready {
			return false
		}
	}
	return true
}

func podHasHighContainerRestarts(pod corev1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.RestartCount >= podRestartAttentionThreshold {
			return true
		}
	}
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.RestartCount >= podRestartAttentionThreshold {
			return true
		}
	}
	return false
}

func podIsBenign(pod corev1.Pod) bool {
	return podIsHealthy(pod) && !podHasHighContainerRestarts(pod)
}

func podNeedsAttention(pod corev1.Pod) bool {
	return !podIsBenign(pod)
}
