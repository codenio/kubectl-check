package plugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestPersistentVolumeClaimNeedsAttention(t *testing.T) {
	cases := []struct {
		name  string
		phase corev1.PersistentVolumeClaimPhase
		want  bool
	}{
		{name: "bound", phase: corev1.ClaimBound, want: false},
		{name: "pending", phase: corev1.ClaimPending, want: true},
		{name: "lost", phase: corev1.ClaimLost, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pvc := corev1.PersistentVolumeClaim{Status: corev1.PersistentVolumeClaimStatus{Phase: tc.phase}}
			if got := persistentVolumeClaimNeedsAttention(pvc); got != tc.want {
				t.Fatalf("persistentVolumeClaimNeedsAttention() = %v, want %v", got, tc.want)
			}
		})
	}
}
