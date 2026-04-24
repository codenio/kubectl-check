package plugin

import (
	"testing"
)

func TestAuditMetav1ListOptions(t *testing.T) {
	t.Run("label only", func(t *testing.T) {
		o := AuditOptions{LabelSelector: "app=web"}
		lo := auditMetav1ListOptions(o)
		if lo.LabelSelector != "app=web" {
			t.Fatalf("LabelSelector = %q", lo.LabelSelector)
		}
		if lo.FieldSelector != "" {
			t.Fatalf("FieldSelector = %q, want empty", lo.FieldSelector)
		}
	})
	t.Run("pod name adds field selector", func(t *testing.T) {
		o := AuditOptions{LabelSelector: "app=web", PodName: "my-pod"}
		lo := auditMetav1ListOptions(o)
		if lo.LabelSelector != "app=web" {
			t.Fatalf("LabelSelector = %q", lo.LabelSelector)
		}
		if lo.FieldSelector != "metadata.name=my-pod" {
			t.Fatalf("FieldSelector = %q", lo.FieldSelector)
		}
	})
}
