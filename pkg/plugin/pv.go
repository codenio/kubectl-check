package plugin

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditPV returns non-Bound persistent volumes as a PersistentVolumeList. benignInScope counts
// Bound volumes (healthy in-use storage).
func AuditPV(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*corev1.PersistentVolumeList, int, int, error) {
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to read kubeconfig: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to create clientset: %w", err)
	}

	pvs, err := clientset.CoreV1().PersistentVolumes().List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list pvs: %w", err)
	}

	totalInScope := len(pvs.Items)
	benignInScope := 0
	filtered := make([]corev1.PersistentVolume, 0, totalInScope)
	for i := range pvs.Items {
		if persistentVolumeNeedsAttention(pvs.Items[i]) {
			filtered = append(filtered, pvs.Items[i])
		} else {
			benignInScope++
		}
	}
	return &corev1.PersistentVolumeList{Items: filtered}, totalInScope, benignInScope, nil
}

func persistentVolumeNeedsAttention(pv corev1.PersistentVolume) bool {
	return pv.Status.Phase != corev1.VolumeBound
}
