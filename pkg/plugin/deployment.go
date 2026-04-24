package plugin

import (
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditDeployments returns Deployments with spec.replicas set to 0 (scaled to zero).
// totalInScope is len(Items) from the unfiltered list; benignInScope counts deployments with
// replicas unset or non-zero.
func AuditDeployments(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*appsv1.DeploymentList, int, int, error) {
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

	list, err := clientset.AppsV1().Deployments(namespace).List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list deployments: %w", err)
	}

	totalInScope := len(list.Items)
	benignInScope := 0
	filtered := make([]appsv1.Deployment, 0)
	for i := range list.Items {
		d := list.Items[i]
		if deploymentScaledToZero(d) {
			filtered = append(filtered, d)
			continue
		}
		benignInScope++
	}
	return &appsv1.DeploymentList{Items: filtered}, totalInScope, benignInScope, nil
}

func deploymentScaledToZero(d appsv1.Deployment) bool {
	r := d.Spec.Replicas
	return r != nil && *r == 0
}
