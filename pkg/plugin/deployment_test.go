package plugin

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestDeploymentScaledToZero(t *testing.T) {
	z := int32(0)
	two := int32(2)
	cases := []struct {
		name string
		d    appsv1.Deployment
		want bool
	}{
		{name: "nil replicas", d: appsv1.Deployment{}, want: false},
		{name: "zero", d: appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &z}}, want: true},
		{name: "non-zero", d: appsv1.Deployment{Spec: appsv1.DeploymentSpec{Replicas: &two}}, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := deploymentScaledToZero(tc.d); got != tc.want {
				t.Fatalf("deploymentScaledToZero() = %v, want %v", got, tc.want)
			}
		})
	}
}
