package plugin

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
)

func TestCronJobSuspended(t *testing.T) {
	tTrue := true
	tFalse := false
	cases := []struct {
		name string
		cj   batchv1.CronJob
		want bool
	}{
		{name: "suspend nil", cj: batchv1.CronJob{}, want: false},
		{name: "suspend false", cj: batchv1.CronJob{Spec: batchv1.CronJobSpec{Suspend: &tFalse}}, want: false},
		{name: "suspend true", cj: batchv1.CronJob{Spec: batchv1.CronJobSpec{Suspend: &tTrue}}, want: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.cj
			if got := cronJobSuspended(&c); got != tc.want {
				t.Fatalf("cronJobSuspended() = %v, want %v", got, tc.want)
			}
		})
	}
}
