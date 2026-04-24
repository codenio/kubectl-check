package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	metav1beta1 "k8s.io/apimachinery/pkg/apis/meta/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	clientscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
)

var auditGVR = map[string]schema.GroupVersionResource{
	"pods":        {Group: "", Version: "v1", Resource: "pods"},
	"nodes":       {Group: "", Version: "v1", Resource: "nodes"},
	"pv":          {Group: "", Version: "v1", Resource: "persistentvolumes"},
	"pvc":         {Group: "", Version: "v1", Resource: "persistentvolumeclaims"},
	"jobs":        {Group: "batch", Version: "v1", Resource: "jobs"},
	"cronjobs":    {Group: "batch", Version: "v1", Resource: "cronjobs"},
	"services":    {Group: "", Version: "v1", Resource: "services"},
	"deployments": {Group: "apps", Version: "v1", Resource: "deployments"},
}

func tableAcceptHeader() string {
	return fmt.Sprintf("application/json;as=Table;v=%s;g=%s,application/json;as=Table;v=%s;g=%s,application/json",
		metav1.SchemeGroupVersion.Version, metav1.GroupName,
		metav1beta1.SchemeGroupVersion.Version, metav1beta1.GroupName,
	)
}

// AsServerTableIfNeeded replaces a typed List with a metav1.Table from the apiserver (same columns as
// kubectl get) when using default or wide output. Other -o formats keep the typed list.
func AsServerTableIfNeeded(cf *genericclioptions.ConfigFlags, resource string, o AuditOptions, obj runtime.Object, outputFormat string) (runtime.Object, error) {
	if outputFormat != "" && outputFormat != "wide" {
		return obj, nil
	}
	if _, ok := auditGVR[resource]; !ok {
		return obj, nil
	}

	cfg, err := cf.ToRESTConfig()
	if err != nil {
		return nil, err
	}
	cs, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, err
	}

	ns, err := namespaceForQuery(cf, o.AllNamespaces)
	if err != nil {
		return nil, err
	}
	// Cluster-scoped APIs (nodes, PVs) must not include a namespace in the path; using the
	// kubeconfig's default namespace would produce .../namespaces/<ns>/nodes and a 404.
	if resource == "nodes" || resource == "pv" {
		ns = ""
	}

	keys, namespaced, empty := objectKeysForFilter(obj, resource)
	opts := auditMetav1ListOptions(o)
	if empty {
		opts = metav1.ListOptions{
			LabelSelector: o.LabelSelector,
			FieldSelector: fmt.Sprintf("metadata.name=%s", "_kubectl_audit_empty_placeholder_"),
		}
	}

	table, err := fetchListAsTable(cs, auditGVR[resource], ns, opts)
	if err != nil {
		return nil, fmt.Errorf("server table print: %w", err)
	}

	// Default table output skips columns with Priority != 0 (see cli-runtime printTable). Some
	// clusters also omit a Namespace column for single-namespace-shaped tables. Fix both when -A.
	if o.AllNamespaces && auditNamespacedResource(resource) {
		ensureNamespaceColumn(table)
	}

	if empty {
		table.Rows = nil
		return table, nil
	}

	filtered := metav1.Table{
		ColumnDefinitions: table.ColumnDefinitions,
		Rows:              make([]metav1.TableRow, 0),
	}
	for i := range table.Rows {
		k := tableRowKey(&table.Rows[i], namespaced)
		if _, keep := keys[k]; keep {
			filtered.Rows = append(filtered.Rows, table.Rows[i])
		}
	}
	return &filtered, nil
}

func auditNamespacedResource(resource string) bool {
	switch resource {
	case "pods", "pvc", "jobs", "cronjobs", "services", "deployments":
		return true
	default:
		return false
	}
}

func ensureNamespaceColumn(table *metav1.Table) {
	hasNS := false
	for i := range table.ColumnDefinitions {
		if strings.EqualFold(table.ColumnDefinitions[i].Name, "Namespace") {
			table.ColumnDefinitions[i].Priority = 0
			hasNS = true
		}
	}
	if hasNS {
		return
	}
	// No Namespace column in Table (e.g. namespaced list path); prepend from object metadata.
	prependNamespaceColumn(table)
}

func prependNamespaceColumn(table *metav1.Table) {
	cols := make([]metav1.TableColumnDefinition, 0, len(table.ColumnDefinitions)+1)
	cols = append(cols, metav1.TableColumnDefinition{Name: "Namespace", Type: "string", Priority: 0})
	cols = append(cols, table.ColumnDefinitions...)
	table.ColumnDefinitions = cols
	for i := range table.Rows {
		ns := namespaceFromRowRaw(table.Rows[i].Object.Raw)
		newCells := make([]interface{}, 0, len(table.Rows[i].Cells)+1)
		newCells = append(newCells, ns)
		newCells = append(newCells, table.Rows[i].Cells...)
		table.Rows[i].Cells = newCells
	}
}

func namespaceFromRowRaw(raw []byte) string {
	if len(raw) == 0 {
		return ""
	}
	var m struct {
		Metadata struct {
			Namespace string `json:"namespace"`
		} `json:"metadata"`
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return ""
	}
	return m.Metadata.Namespace
}

func objectKeysForFilter(obj runtime.Object, resource string) (keys map[string]struct{}, namespaced, empty bool) {
	keys = make(map[string]struct{})
	namespaced = resource != "nodes" && resource != "pv"

	switch list := obj.(type) {
	case *corev1.PodList:
		if len(list.Items) == 0 {
			return keys, namespaced, true
		}
		for _, p := range list.Items {
			keys[p.Namespace+"/"+p.Name] = struct{}{}
		}
	case *corev1.NodeList:
		if len(list.Items) == 0 {
			return keys, false, true
		}
		for _, n := range list.Items {
			keys[n.Name] = struct{}{}
		}
	case *corev1.PersistentVolumeList:
		if len(list.Items) == 0 {
			return keys, false, true
		}
		for _, pv := range list.Items {
			keys[pv.Name] = struct{}{}
		}
	case *corev1.PersistentVolumeClaimList:
		if len(list.Items) == 0 {
			return keys, namespaced, true
		}
		for _, p := range list.Items {
			keys[p.Namespace+"/"+p.Name] = struct{}{}
		}
	case *batchv1.JobList:
		if len(list.Items) == 0 {
			return keys, namespaced, true
		}
		for _, j := range list.Items {
			keys[j.Namespace+"/"+j.Name] = struct{}{}
		}
	case *batchv1.CronJobList:
		if len(list.Items) == 0 {
			return keys, namespaced, true
		}
		for _, c := range list.Items {
			keys[c.Namespace+"/"+c.Name] = struct{}{}
		}
	case *corev1.ServiceList:
		if len(list.Items) == 0 {
			return keys, namespaced, true
		}
		for _, s := range list.Items {
			keys[s.Namespace+"/"+s.Name] = struct{}{}
		}
	case *appsv1.DeploymentList:
		if len(list.Items) == 0 {
			return keys, namespaced, true
		}
		for _, d := range list.Items {
			keys[d.Namespace+"/"+d.Name] = struct{}{}
		}
	default:
		return keys, namespaced, true
	}
	return keys, namespaced, false
}

func tableRowKey(row *metav1.TableRow, namespaced bool) string {
	if row.Object.Object != nil {
		if a, err := meta.Accessor(row.Object.Object); err == nil {
			if namespaced {
				return a.GetNamespace() + "/" + a.GetName()
			}
			return a.GetName()
		}
	}
	if len(row.Object.Raw) > 0 {
		var wrap struct {
			Metadata metav1.ObjectMeta `json:"metadata"`
		}
		if err := json.Unmarshal(row.Object.Raw, &wrap); err == nil {
			if namespaced {
				return wrap.Metadata.Namespace + "/" + wrap.Metadata.Name
			}
			return wrap.Metadata.Name
		}
	}
	return ""
}

func fetchListAsTable(cs kubernetes.Interface, gvr schema.GroupVersionResource, namespace string, opts metav1.ListOptions) (*metav1.Table, error) {
	var restClient rest.Interface
	switch gvr.Group {
	case "":
		restClient = cs.CoreV1().RESTClient()
	case "batch":
		restClient = cs.BatchV1().RESTClient()
	case "apps":
		restClient = cs.AppsV1().RESTClient()
	default:
		return nil, fmt.Errorf("unsupported API group %q", gvr.Group)
	}

	req := restClient.Get().Resource(gvr.Resource)
	if namespace != "" {
		req = req.Namespace(namespace)
	}

	table := &metav1.Table{}
	err := req.VersionedParams(&opts, clientscheme.ParameterCodec).
		SetHeader("Accept", tableAcceptHeader()).
		Do(context.Background()).
		Into(table)
	if err != nil {
		return nil, err
	}
	return table, nil
}
