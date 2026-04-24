// Package plugin implements kubectl audit resource listing and filtering.
// Audit logic lives in per-resource files (e.g. pods.go, jobs.go, service.go, deployment.go).
package plugin

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

// AuditOptions configures listing and filtering for audit commands.
type AuditOptions struct {
	AllNamespaces bool
	LabelSelector string
	// PodName, if set, restricts listing to pods with this exact metadata.name (used by audit containers).
	PodName string
}

func namespaceForQuery(configFlags *genericclioptions.ConfigFlags, allNamespaces bool) (string, error) {
	if allNamespaces {
		return "", nil
	}

	namespace, _, err := configFlags.ToRawKubeConfigLoader().Namespace()
	if err != nil {
		return "", fmt.Errorf("failed to resolve namespace: %w", err)
	}
	return namespace, nil
}

func auditMetav1ListOptions(o AuditOptions) metav1.ListOptions {
	lo := metav1.ListOptions{LabelSelector: o.LabelSelector}
	if o.PodName != "" {
		lo.FieldSelector = fields.OneTermEqualSelector("metadata.name", o.PodName).String()
	}
	return lo
}
