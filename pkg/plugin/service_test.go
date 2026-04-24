package plugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestServiceHasMatchingPod(t *testing.T) {
	sel := labels.Set(map[string]string{"app": "web"}).AsSelectorPreValidated()
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "api"}}},
		{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": "web", "tier": "fe"}}},
	}
	if serviceHasMatchingPod(pods[:1], sel) {
		t.Fatal("expected no match")
	}
	if !serviceHasMatchingPod(pods, sel) {
		t.Fatal("expected match")
	}
}

func TestServiceExemptFromPodSelectorAudit(t *testing.T) {
	if !serviceExemptFromPodSelectorAudit(corev1.Service{
		Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeExternalName},
	}) {
		t.Fatal("expected ExternalName exempt")
	}
	if !serviceExemptFromPodSelectorAudit(corev1.Service{
		Spec: corev1.ServiceSpec{Type: corev1.ServiceTypeClusterIP},
	}) {
		t.Fatal("expected empty selector exempt")
	}
	if serviceExemptFromPodSelectorAudit(corev1.Service{
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: map[string]string{"app": "x"},
		},
	}) {
		t.Fatal("expected ClusterIP with selector not exempt")
	}
}

func TestGroupPodsByNamespace(t *testing.T) {
	pods := []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "p1"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "b", Name: "p2"}},
		{ObjectMeta: metav1.ObjectMeta{Namespace: "a", Name: "p3"}},
	}
	m := groupPodsByNamespace(pods)
	if len(m["a"]) != 2 || len(m["b"]) != 1 {
		t.Fatalf("unexpected grouping: %#v", m)
	}
}
