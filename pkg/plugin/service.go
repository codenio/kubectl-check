package plugin

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditServices returns Services whose pod selector matches no pods in the same namespace.
// ExternalName services and services with an empty selector are treated as in scope but benign
// (they do not express a pod routing expectation via selector).
func AuditServices(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*corev1.ServiceList, int, int, error) {
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

	listOpts := auditMetav1ListOptions(o)
	svcs, err := clientset.CoreV1().Services(namespace).List(context.Background(), listOpts)
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list services: %w", err)
	}

	// Match the full pod set in each namespace; the audit label selector applies to Services only.
	pods, err := clientset.CoreV1().Pods(namespace).List(context.Background(), metav1.ListOptions{})
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list pods: %w", err)
	}

	podsByNS := groupPodsByNamespace(pods.Items)

	totalInScope := len(svcs.Items)
	benignInScope := 0
	filtered := make([]corev1.Service, 0)
	for i := range svcs.Items {
		svc := svcs.Items[i]
		if serviceExemptFromPodSelectorAudit(svc) {
			benignInScope++
			continue
		}
		sel := labels.Set(svc.Spec.Selector).AsSelectorPreValidated()
		nsPods := podsByNS[svc.Namespace]
		if serviceHasMatchingPod(nsPods, sel) {
			benignInScope++
			continue
		}
		filtered = append(filtered, svc)
	}
	return &corev1.ServiceList{Items: filtered}, totalInScope, benignInScope, nil
}

// serviceExemptFromPodSelectorAudit is true for ExternalName and empty-selector Services (no
// pod-based selector expectation for this audit).
func serviceExemptFromPodSelectorAudit(svc corev1.Service) bool {
	return svc.Spec.Type == corev1.ServiceTypeExternalName || len(svc.Spec.Selector) == 0
}

func groupPodsByNamespace(pods []corev1.Pod) map[string][]corev1.Pod {
	out := make(map[string][]corev1.Pod)
	for i := range pods {
		p := pods[i]
		out[p.Namespace] = append(out[p.Namespace], p)
	}
	return out
}

func serviceHasMatchingPod(pods []corev1.Pod, sel labels.Selector) bool {
	for i := range pods {
		if sel.Matches(labels.Set(pods[i].Labels)) {
			return true
		}
	}
	return false
}
