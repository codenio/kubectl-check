package plugin

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditNodes returns NotReady or unschedulable nodes as a NodeList. benignInScope counts nodes that
// are Ready and schedulable (healthy).
func AuditNodes(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*corev1.NodeList, int, int, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create clientset: %w", err)
	}

	nodes, err := clientset.CoreV1().Nodes().List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list nodes: %w", err)
	}

	totalInScope := len(nodes.Items)
	benignInScope := 0
	filtered := make([]corev1.Node, 0, totalInScope)
	for i := range nodes.Items {
		if nodeIsHealthy(nodes.Items[i]) {
			benignInScope++
		} else {
			filtered = append(filtered, nodes.Items[i])
		}
	}
	return &corev1.NodeList{Items: filtered}, totalInScope, benignInScope, nil
}

func nodeIsHealthy(node corev1.Node) bool {
	return !isNotReadyNode(node) && !node.Spec.Unschedulable
}

func isNotReadyNode(node corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status != corev1.ConditionTrue
		}
	}
	return true
}
