package plugin

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestPodNeedsAttention(t *testing.T) {
	cases := []struct {
		name string
		pod  corev1.Pod
		want bool
	}{
		{
			name: "running all ready",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true},
					},
				},
			},
			want: false,
		},
		{
			name: "running one not ready",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true},
						{Ready: false},
					},
				},
			},
			want: true,
		},
		{
			name: "pending",
			pod: corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodPending},
			},
			want: true,
		},
		{
			name: "succeeded",
			pod: corev1.Pod{
				Status: corev1.PodStatus{Phase: corev1.PodSucceeded},
			},
			want: true,
		},
		{
			name: "running all ready restarts below threshold",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, RestartCount: 4},
					},
				},
			},
			want: false,
		},
		{
			name: "running all ready high restarts on main container",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, RestartCount: 5},
					},
				},
			},
			want: true,
		},
		{
			name: "running all ready high restarts on init only",
			pod: corev1.Pod{
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
					ContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, RestartCount: 0},
					},
					InitContainerStatuses: []corev1.ContainerStatus{
						{Ready: true, RestartCount: 5},
					},
				},
			},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := podNeedsAttention(tc.pod); got != tc.want {
				t.Fatalf("podNeedsAttention() = %v, want %v", got, tc.want)
			}
		})
	}
}
