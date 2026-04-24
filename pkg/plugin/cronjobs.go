package plugin

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

// AuditCronJobs returns suspended cron jobs as a CronJobList (batch/v1). benignInScope counts
// CronJobs that are not suspended (desired to run on schedule).
func AuditCronJobs(configFlags *genericclioptions.ConfigFlags, o AuditOptions) (*batchv1.CronJobList, int, int, error) {
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

	cronJobs, err := clientset.BatchV1().CronJobs(namespace).List(context.Background(), auditMetav1ListOptions(o))
	if err != nil {
		return nil, 0, 0, fmt.Errorf("failed to list cronjobs: %w", err)
	}

	totalInScope := len(cronJobs.Items)
	benignInScope := 0
	filtered := make([]batchv1.CronJob, 0, totalInScope)
	for i := range cronJobs.Items {
		if cronJobSuspended(&cronJobs.Items[i]) {
			filtered = append(filtered, cronJobs.Items[i])
		} else {
			benignInScope++
		}
	}
	return &batchv1.CronJobList{Items: filtered}, totalInScope, benignInScope, nil
}

func cronJobSuspended(c *batchv1.CronJob) bool {
	s := c.Spec.Suspend
	return s != nil && *s
}
