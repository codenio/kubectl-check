package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/codenio/kubectl-audit/pkg/plugin"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/kubectl/pkg/cmd/get"
)

var (
	KubernetesConfigFlags *genericclioptions.ConfigFlags
	// AuditPrintFlags is kubectl get's PrintFlags (k8s.io/kubectl/pkg/cmd/get).
	AuditPrintFlags *get.PrintFlags
)

func RootCmd() *cobra.Command {
	AuditPrintFlags = NewAuditPrintFlags()

	root := &cobra.Command{
		Use:   "audit",
		Short: "Run cluster audits with kubectl-compatible output",
		Long: `Lists resources that fail common health checks, with kubectl get-style output (-o wide, json, yaml, etc.) 
and a one-line summary before the table. 

  Find more information at: https://github.com/codenio/kubectl-audit/blob/main/README.md
`,
		Example: `  kubectl audit containers
  kubectl audit containers --pod my-pod-abc
  kubectl audit pods
  kubectl audit pods -o wide
  kubectl audit service
  kubectl audit deploy -A
  kubectl audit nodes -o json
  kubectl audit pvc -o yaml
  kubectl audit jobs -o custom-columns=NAME:.metadata.name
  kubectl audit pods -A --selector app=nginx`,
		SilenceErrors: true,
		SilenceUsage:  true,
		// Cobra adds a "completion" subcommand by default; kubectl already has `kubectl completion`.
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	// Default Cobra order is Examples then Available Commands; show subcommands first.
	root.SetUsageTemplate(auditRootUsageTemplate())

	root.PersistentFlags().BoolP("all-namespaces", "A", false, "If true, check the specified resource across all namespaces")
	root.PersistentFlags().StringP("selector", "l", "", "Selector (label query) to filter on, supports '=', '==', and '!='")

	KubernetesConfigFlags = genericclioptions.NewConfigFlags(true)
	KubernetesConfigFlags.AddFlags(root.PersistentFlags())

	for _, def := range auditResourceCommands {
		root.AddCommand(newAuditResourceCmd(def))
	}

	return root
}

type auditResourceCmdDef struct {
	resource  string
	use       string
	short     string
	aliases   []string
	podFilter bool // register --pod / -p (containers only)
}

var auditResourceCommands = []auditResourceCmdDef{
	{resource: "containers", use: "containers", short: "Audit containers (derived from pods)", aliases: []string{"container"}, podFilter: true},
	{resource: "pods", use: "pods", short: "Audit pods", aliases: []string{"pod", "po"}},
	{resource: "nodes", use: "nodes", short: "Audit nodes", aliases: []string{"node", "no"}},
	{resource: "pv", use: "pv", short: "Audit persistent volumes", aliases: []string{"pvs", "persistentvolume", "persistentvolumes"}},
	{resource: "pvc", use: "pvc", short: "Audit persistent volume claims", aliases: []string{"pvcs", "persistentvolumeclaim", "persistentvolumeclaims"}},
	{resource: "jobs", use: "jobs", short: "Audit jobs", aliases: []string{"job"}},
	{resource: "cronjobs", use: "cronjobs", short: "Audit cron jobs", aliases: []string{"cronjob", "cj"}},
	{resource: "services", use: "service", short: "Audit services whose selectors match no pods", aliases: []string{"services", "svc"}},
	{resource: "deployments", use: "deploy", short: "Audit deployments scaled to zero replicas", aliases: []string{"deployment", "deployments"}},
}

func newAuditResourceCmd(def auditResourceCmdDef) *cobra.Command {
	res := def.resource
	cmd := &cobra.Command{
		Use:     def.use,
		Aliases: def.aliases,
		Short:   def.short,
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAudit(res, cmd)
		},
	}
	if def.podFilter {
		cmd.Use = "containers [-p POD|--pod POD]"
		cmd.Flags().StringP("pod", "p", "", "only for containers)")
		_ = cmd.Flags().MarkHidden("pod")
		cmd.Long = def.short
	}
	AuditPrintFlags.AddFlags(cmd)
	return cmd
}

func runAudit(res string, cmd *cobra.Command) error {
	allNS, _ := cmd.PersistentFlags().GetBool("all-namespaces")
	if !allNS {
		allNS = allNamespacesInArgv()
	}
	sel, _ := cmd.PersistentFlags().GetString("selector")

	opts := plugin.AuditOptions{
		AllNamespaces: allNS,
		LabelSelector: sel,
	}
	if res == "containers" {
		pod, _ := cmd.Flags().GetString("pod")
		opts.PodName = strings.TrimSpace(pod)
	}

	gk, ok := auditGroupKinds[res]
	if !ok {
		return fmt.Errorf("unknown resource %q", res)
	}

	var (
		obj           runtime.Object
		totalInScope  int
		benignInScope int
		err           error
	)
	switch res {
	case "containers":
		obj, totalInScope, benignInScope, err = plugin.AuditContainers(KubernetesConfigFlags, opts)
	case "pods":
		obj, totalInScope, benignInScope, err = plugin.AuditPods(KubernetesConfigFlags, opts)
	case "nodes":
		obj, totalInScope, benignInScope, err = plugin.AuditNodes(KubernetesConfigFlags, opts)
	case "pv":
		obj, totalInScope, benignInScope, err = plugin.AuditPV(KubernetesConfigFlags, opts)
	case "pvc":
		obj, totalInScope, benignInScope, err = plugin.AuditPVC(KubernetesConfigFlags, opts)
	case "jobs":
		obj, totalInScope, benignInScope, err = plugin.AuditJobs(KubernetesConfigFlags, opts)
	case "cronjobs":
		obj, totalInScope, benignInScope, err = plugin.AuditCronJobs(KubernetesConfigFlags, opts)
	case "services":
		obj, totalInScope, benignInScope, err = plugin.AuditServices(KubernetesConfigFlags, opts)
	case "deployments":
		obj, totalInScope, benignInScope, err = plugin.AuditDeployments(KubernetesConfigFlags, opts)
	default:
		return fmt.Errorf("unknown resource %q", res)
	}
	if err != nil {
		return err
	}

	out := ""
	if AuditPrintFlags.OutputFormat != nil {
		out = *AuditPrintFlags.OutputFormat
	}
	if AuditPrintFlags.TemplateFlags != nil &&
		AuditPrintFlags.TemplateFlags.TemplateArgument != nil &&
		len(*AuditPrintFlags.TemplateFlags.TemplateArgument) > 0 && out == "" {
		out = "go-template"
	}

	if isHumanTableOutput(out) && auditObjectLen(obj) == 0 {
		total, benign, attention := plugin.SummarizeAudit(totalInScope, benignInScope, obj)
		WriteAuditSummary(os.Stdout, auditSummaryResourceTitle(res), total, benign, attention)
		writeAuditEmptyMessage(os.Stderr, res, allNS, KubernetesConfigFlags, totalInScope)
		return nil
	}

	total, benign, attention := plugin.SummarizeAudit(totalInScope, benignInScope, obj)

	obj, err = plugin.AsServerTableIfNeeded(KubernetesConfigFlags, res, opts, obj, out)
	if err != nil {
		return err
	}

	summaryDest := os.Stderr
	if isHumanTableOutput(out) {
		summaryDest = os.Stdout
	}
	WriteAuditSummary(summaryDest, auditSummaryResourceTitle(res), total, benign, attention)

	if res == "containers" {
		cl, ok := obj.(*plugin.ContainerList)
		if !ok {
			return fmt.Errorf("internal error: expected *plugin.ContainerList")
		}
		return printContainersOutput(cl, out, out == "wide", withNamespaceColumn(res, allNS))
	}

	withNS := withNamespaceColumn(res, allNS)
	// Server Tables already include a NAMESPACE column when listing cluster-wide. Setting
	// PrintOptions.WithNamespace makes decorateTable prepend another column from row metadata;
	// row.Object is often unset (only Raw), so that prepends an empty cell and shifts columns.
	if _, ok := obj.(*metav1.Table); ok {
		withNS = false
	}
	return printObjects(obj, AuditPrintFlags, gk, withNS, isHumanTableOutput(out))
}

var auditGroupKinds = map[string]schema.GroupKind{
	"containers":  {Group: "", Kind: "Container"},
	"pods":        {Group: "", Kind: "Pod"},
	"nodes":       {Group: "", Kind: "Node"},
	"pv":          {Group: "", Kind: "PersistentVolume"},
	"pvc":         {Group: "", Kind: "PersistentVolumeClaim"},
	"jobs":        {Group: "batch", Kind: "Job"},
	"cronjobs":    {Group: "batch", Kind: "CronJob"},
	"services":    {Group: "", Kind: "Service"},
	"deployments": {Group: "apps", Kind: "Deployment"},
}

func auditSummaryResourceTitle(resource string) string {
	if gk, ok := auditGroupKinds[resource]; ok {
		return gk.Kind
	}
	return "Resource"
}

// allNamespacesInArgv catches -A / --all-namespaces if pflag/cobra did not (e.g. flag order with
// kubectl get PrintFlags or PersistentFlags merges).
func allNamespacesInArgv() bool {
	for _, a := range os.Args[1:] {
		switch a {
		case "-A", "--all-namespaces":
			return true
		}
		if strings.HasPrefix(a, "--all-namespaces=") {
			v := strings.TrimPrefix(a, "--all-namespaces=")
			return v == "" || v == "true" || v == "1" || v == "True"
		}
	}
	return false
}

func withNamespaceColumn(resource string, allNS bool) bool {
	if !allNS {
		return false
	}
	switch resource {
	case "containers", "pods", "pvc", "jobs", "cronjobs", "services", "deployments":
		return true
	default:
		return false
	}
}

func isHumanTableOutput(out string) bool {
	return out == "" || out == "wide"
}

func auditObjectLen(obj runtime.Object) int {
	if t, ok := obj.(*metav1.Table); ok {
		return len(t.Rows)
	}
	return meta.LenList(obj)
}

// writeAuditEmptyMessage prints a line after an empty audit table. inScopeCount is how many
// resources were listed before filtering (same basis as the summary total). When it is zero,
// nothing exists in scope; when positive, resources exist but none match the audit.
func writeAuditEmptyMessage(w io.Writer, resource string, allNS bool, cf *genericclioptions.ConfigFlags, inScopeCount int) {
	phrase := auditResourcePhrase(resource)
	namespaced := resource == "containers" || resource == "pods" || resource == "pvc" || resource == "jobs" || resource == "cronjobs" || resource == "services" || resource == "deployments"

	if inScopeCount > 0 {
		if namespaced && !allNS {
			ns, _, err := cf.ToRawKubeConfigLoader().Namespace()
			if err != nil || ns == "" {
				fmt.Fprintf(w, "No %s require attention.\n", phrase)
				return
			}
			fmt.Fprintf(w, "No %s require attention in %s namespace.\n", phrase, ns)
			return
		}
		fmt.Fprintf(w, "No %s require attention.\n", phrase)
		return
	}

	if namespaced && !allNS {
		ns, _, err := cf.ToRawKubeConfigLoader().Namespace()
		if err != nil || ns == "" {
			fmt.Fprintf(w, "No %s found.\n", phrase)
			return
		}
		fmt.Fprintf(w, "No %s found in %s namespace.\n", phrase, ns)
		return
	}
	fmt.Fprintf(w, "No %s found.\n", phrase)
}

func auditResourcePhrase(resource string) string {
	switch resource {
	case "containers":
		return "containers"
	case "pods":
		return "pods"
	case "nodes":
		return "nodes"
	case "pv":
		return "persistent volumes"
	case "pvc":
		return "persistent volume claims"
	case "jobs":
		return "jobs"
	case "cronjobs":
		return "cron jobs"
	case "services":
		return "services"
	case "deployments":
		return "deployments"
	default:
		return "resources"
	}
}

// auditRootUsageTemplate matches Cobra's default usage template except Examples follow Available Commands.
func auditRootUsageTemplate() string {
	return `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if gt (len .Aliases) 0}}

Aliases:
  {{.NameAndAliases}}{{end}}{{if .HasAvailableSubCommands}}{{$cmds := .Commands}}{{if eq (len .Groups) 0}}

Available Commands:{{range $cmds}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{else}}{{range $group := .Groups}}

{{.Title}}{{range $cmds}}{{if (and (eq .GroupID $group.ID) (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if not .AllChildCommandsHaveGroup}}

Additional Commands:{{range $cmds}}{{if (and (eq .GroupID "") (or .IsAvailableCommand (eq .Name "help")))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{end}}{{end}}{{if .HasExample}}

Examples:
{{.Example}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
}

func InitAndExecute() {
	if err := RootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
