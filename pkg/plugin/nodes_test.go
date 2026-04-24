package plugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestIsNotReadyNode(t *testing.T) {
	cases := []struct {
		name string
		node corev1.Node
		want bool
	}{
		{
			name: "ready true",
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
					},
				},
			},
			want: false,
		},
		{
			name: "ready false",
			node: corev1.Node{
				Status: corev1.NodeStatus{
					Conditions: []corev1.NodeCondition{
						{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					},
				},
			},
			want: true,
		},
		{
			name: "no ready condition",
			node: corev1.Node{Status: corev1.NodeStatus{Conditions: nil}},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isNotReadyNode(tc.node); got != tc.want {
				t.Fatalf("isNotReadyNode() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNodeIsHealthy(t *testing.T) {
	ready := corev1.Node{
		Spec: corev1.NodeSpec{Unschedulable: false},
		Status: corev1.NodeStatus{
			Conditions: []corev1.NodeCondition{
				{Type: corev1.NodeReady, Status: corev1.ConditionTrue},
			},
		},
	}
	if !nodeIsHealthy(ready) {
		t.Fatal("expected healthy ready node")
	}
	cordoned := ready.DeepCopy()
	cordoned.Spec.Unschedulable = true
	if nodeIsHealthy(*cordoned) {
		t.Fatal("expected cordoned node not healthy")
	}
	notReady := ready.DeepCopy()
	notReady.Status.Conditions[0].Status = corev1.ConditionFalse
	if nodeIsHealthy(*notReady) {
		t.Fatal("expected NotReady node not healthy")
	}
}
