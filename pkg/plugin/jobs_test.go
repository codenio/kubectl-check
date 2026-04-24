package plugin

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestIsProblemJob(t *testing.T) {
	cases := []struct {
		name string
		job  batchv1.Job
		want bool
	}{
		{
			name: "no problems",
			job:  batchv1.Job{Status: batchv1.JobStatus{Failed: 0}},
			want: false,
		},
		{
			name: "status failed count",
			job:  batchv1.Job{Status: batchv1.JobStatus{Failed: 1}},
			want: true,
		},
		{
			name: "job failed condition",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Type: batchv1.JobFailed, Status: corev1.ConditionTrue},
					},
				},
			},
			want: true,
		},
		{
			name: "backoff exceeded",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Reason: "BackoffLimitExceeded"},
					},
				},
			},
			want: true,
		},
		{
			name: "deadline exceeded",
			job: batchv1.Job{
				Status: batchv1.JobStatus{
					Conditions: []batchv1.JobCondition{
						{Reason: "DeadlineExceeded"},
					},
				},
			},
			want: true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isProblemJob(tc.job); got != tc.want {
				t.Fatalf("isProblemJob() = %v, want %v", got, tc.want)
			}
		})
	}
}
