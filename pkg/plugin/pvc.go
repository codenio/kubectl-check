package plugin

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditPVC returns non-Bound claims as a PersistentVolumeClaimList. benignInScope counts Bound claims.
func AuditPVC(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*corev1.PersistentVolumeClaimList, int, int, error) {
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

	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list pvcs: %w", err)
	}

	totalInScope := len(pvcs.Items)
	benignInScope := 0
	filtered := make([]corev1.PersistentVolumeClaim, 0, totalInScope)
	for i := range pvcs.Items {
		if persistentVolumeClaimNeedsAttention(pvcs.Items[i]) {
			filtered = append(filtered, pvcs.Items[i])
		} else {
			benignInScope++
		}
	}
	return &corev1.PersistentVolumeClaimList{Items: filtered}, totalInScope, benignInScope, nil
}

func persistentVolumeClaimNeedsAttention(pvc corev1.PersistentVolumeClaim) bool {
	return pvc.Status.Phase != corev1.ClaimBound
}
