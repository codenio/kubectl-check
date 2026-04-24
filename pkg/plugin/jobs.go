package plugin

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditJobs returns failed or backoff/deadline problem jobs as a JobList. benignInScope counts jobs
// with no failure/backoff/deadline problems.
func AuditJobs(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*batchv1.JobList, int, int, error) {
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

	jobs, err := clientset.BatchV1().Jobs(namespace).List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list jobs: %w", err)
	}

	totalInScope := len(jobs.Items)
	benignInScope := 0
	filtered := make([]batchv1.Job, 0, totalInScope)
	for i := range jobs.Items {
		if !isProblemJob(jobs.Items[i]) {
			benignInScope++
		} else {
			filtered = append(filtered, jobs.Items[i])
		}
	}
	return &batchv1.JobList{Items: filtered}, totalInScope, benignInScope, nil
}

func isProblemJob(job batchv1.Job) bool {
	if job.Status.Failed > 0 {
		return true
	}

	for _, c := range job.Status.Conditions {
		if c.Type == batchv1.JobFailed && c.Status == corev1.ConditionTrue {
			return true
		}
		if c.Reason == "BackoffLimitExceeded" || c.Reason == "DeadlineExceeded" {
			return true
		}
	}
	return false
}
