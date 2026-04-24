package plugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestPersistentVolumeNeedsAttention(t *testing.T) {
	cases := []struct {
		name  string
		phase corev1.PersistentVolumePhase
		want  bool
	}{
		{name: "bound", phase: corev1.VolumeBound, want: false},
		{name: "available", phase: corev1.VolumeAvailable, want: true},
		{name: "pending", phase: corev1.VolumePending, want: true},
		{name: "released", phase: corev1.VolumeReleased, want: true},
		{name: "failed", phase: corev1.VolumeFailed, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pv := corev1.PersistentVolume{Status: corev1.PersistentVolumeStatus{Phase: tc.phase}}
			if got := persistentVolumeNeedsAttention(pv); got != tc.want {
				t.Fatalf("persistentVolumeNeedsAttention() = %v, want %v", got, tc.want)
			}
		})
	}
}
