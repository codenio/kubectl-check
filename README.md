# kubectl-audit

**Kubernetes `kubectl` plugin for cluster health: find unhealthy pods, container issues, nodes, storage, batch workloads, Services with no backing Pods, and Deployments scaled to zero.**

[`kubectl-audit`](https://github.com/codenio/kubectl-audit) is a [`kubectl` plugin](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/) ([Krew](#install)) that surfaces resources failing common checks—**pods** that are not fully healthy (including high **restart counts** and bad phases such as **CrashLoopBackOff** or **ImagePullBackOff**), **containers** as individual rows (init and app, derived from pods), **nodes** that are **NotReady** or **cordoned** (**SchedulingDisabled**), **PersistentVolumes** (PV) and **PersistentVolumeClaims** (PVC) not **Bound**, **failed Jobs**, **suspended CronJobs**, **Services** whose **pod selector matches no Pods** in the same namespace (skipping **ExternalName** and empty selectors), and **Deployments** with **`spec.replicas` set to 0**. For most kinds, output matches [`kubectl get`](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_get/) printers (default table, `-o wide`, JSON, YAML, custom columns, Go templates). The **containers** subcommand uses a dedicated table and supports `-o json`, `-o yaml`, `-o name`, default table, and `-o wide` (see [Output formats](#output-formats)).

Use it for **Kubernetes troubleshooting**, **SRE / platform** triage, and **pre-deploy smoke checks** without leaving the CLI.

## Contents

- [What it checks](#what-it-checks)
- [Install](#install)
- [Usage](#usage)
- [Resources and filters](#resources-and-filters)
- [Output formats](#output-formats)
- [Examples](#examples)
  - [Example manifests by audit](#example-manifests-by-audit)
  - [Sample output](#sample-output)
- [Development](#development)
- [Contributing](#contributing)
- [Acknowledgments](#acknowledgments)

## What it checks

| Area | Plain-language intent |
| ---- | ---------------------- |
| **Pods** | Pods that are not in a good steady state or have risky restart behavior (see [Resources and filters](#resources-and-filters)). |
| **Containers** | Per-container problems (waits, pull errors, readiness, high restarts) with optional filter by pod name. |
| **Nodes** | Nodes you cannot schedule to or that are not ready. |
| **Storage (PV / PVC)** | Volumes and claims stuck outside **Bound**. |
| **Jobs / CronJobs** | Failed jobs and cron jobs that are **suspended**. |
| **Services** | Services with a **non-empty** pod **selector** and **no** matching **Pods** in that namespace (**ExternalName** and empty selectors are skipped). |
| **Deployments** | Deployments whose **`spec.replicas`** is explicitly **0** (scaled to zero; `nil` replicas is treated as not in scope for this check). |

## Install

### Krew (codenio custom index)

Published through [codenio-krew-index](https://github.com/codenio/codenio-krew-index):

```bash
kubectl krew index add codenio https://github.com/codenio/codenio-krew-index.git
kubectl krew install codenio/audit
```

Upgrade when the index is updated (Krew accepts **only the short plugin name** here — not `codenio/audit`):

```bash
kubectl krew upgrade audit
```

If the local index name `codenio` is already taken, pick another name for `kubectl krew index add` and use it in **install** only (for example `kubectl krew install codenio-krew/audit`). **Upgrade** stays `kubectl krew upgrade audit` regardless of which index you installed from. Maintainer notes for bumping versions live in the index repository.

### Krew (default index)

If the plugin is listed in Krew’s default index:

```bash
kubectl krew install audit
```

### From source

Requires [Go](https://go.dev/dl/) 1.21+ and `make`:

```bash
git clone https://github.com/codenio/kubectl-audit.git
cd kubectl-audit
make install
```

This builds `bin/audit` and copies it to `~/.krew/bin/kubectl-audit`. Put `~/.krew/bin` on your `PATH` (or adjust the install path) so `kubectl audit` resolves.

## Usage

The plugin is a single binary with **subcommands** per audit target (see [Resources and filters](#resources-and-filters)):

```bash
kubectl audit --help
kubectl audit pods
kubectl audit containers
```

Standard `kubectl` config applies: current context, `KUBECONFIG`, `-n` / `--namespace`, `--context`, and so on.

## Resources and filters


| Subcommand   | Aliases                                                   | What is listed                                                                                                                                                                                                                                                                        |
| ------------ | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `containers` | `container`                                               | **Init and app** container rows that need attention (image pull / crash-style waiting reasons, high restarts, not ready when the pod is not in a terminal phase, failed init, etc.). Rows include **POD** (pod name) and **NAME** (container name). Use **`-p` / `--pod`** with an exact pod `metadata.name` to scope to one pod (see `kubectl audit containers --help` Usage line). |
| `pods`       | `pod`, `po`                                               | Pods that need attention: phase is not `Running`, any regular container is not `Ready`, **or** any regular or init container has `RestartCount` ≥ **5** (threshold is fixed in code). `Succeeded` / `Completed` job pods are included because they are not in a running steady state. |
| `nodes`      | `node`, `no`                                              | Nodes that are `NotReady` or have `SchedulingDisabled`.                                                                                                                                                                                                                               |
| `pvc`        | `pvcs`, `persistentvolumeclaim`, `persistentvolumeclaims` | PVCs not in `Bound` phase.                                                                                                                                                                                                                                                            |
| `pv`         | `pvs`, `persistentvolume`, `persistentvolumes`            | PVs not in `Bound` phase.                                                                                                                                                                                                                                                             |
| `jobs`       | `job`                                                     | Failed jobs (including backoff / deadline failures).                                                                                                                                                                                                                                  |
| `cronjobs`   | `cronjob`, `cj`                                           | Suspended cron jobs.                                                                                                                                                                                                                                                                  |
| `service`    | `services`, `svc`                                         | Services whose **selector** matches **no Pods** in the namespace (**ExternalName** and empty **selector** are out of scope for this check). The audit **`-l` / `--selector`** filters **Services** only; Pod matching uses all Pods in each namespace.                               |
| `deploy`     | `deployment`, `deployments`                               | Deployments with **`spec.replicas: 0`** (scaled to zero).                                                                                                                                                                                                                            |


**Common flags**

- **All namespaces:** `-A` or `--all-namespaces` for namespaced targets (`containers`, `pods`, `pvc`, `jobs`, `cronjobs`, `service`, `deploy`).
- **Labels:** `-l` / `--selector` (same semantics as `kubectl get`; applies to the underlying pod list for `containers`, to the **Service** list for `kubectl audit service`, and to the **Deployment** list for `kubectl audit deploy`).

There are no `--pending` / `--failed` style switches: one `kubectl audit pods` run applies all pod rules above; `kubectl audit containers` applies per-container rules.

Further notes live in [doc/USAGE.md](doc/USAGE.md).

## Output formats

For **pods**, **nodes**, **pv**, **pvc**, **jobs**, **cronjobs**, **service**, and **deploy**, [`kubectl get`](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_get/)-style `-o` flags work as usual, for example:

```bash
kubectl audit pods -o wide
kubectl audit nodes -o json
kubectl audit pvc -o yaml
kubectl audit jobs -o custom-columns=NAME:.metadata.name
kubectl audit service -o json
```

For **`containers`**, printing is custom: **default** and **`-o wide`** use a fixed column layout. Default columns are **NAMESPACE** (with `-A`), **POD**, **NAME**, **READY**, **STATUS**, **RESTARTS**, **AGE**, **TYPE** (`container` vs `init-container`). **`-o wide`** adds **PORTS**, **IMAGE**, and **PULLPOLICY**. Machine output: **`-o json`**, **`-o yaml`**, **`-o name`** only (other `-o` values are rejected with a clear error).

## Examples

```bash
# Containers: per-container rows (pod name first column; use -o wide for image, ports, pull policy)
kubectl audit containers
kubectl audit containers -A
kubectl audit containers -o wide
kubectl audit containers -p my-pod-0
kubectl audit containers --pod my-pod-0 -n my-namespace

# Pods: current context default namespace
kubectl audit pods

# Pods: single explicit namespace (no NAMESPACE column in the default table, same as kubectl get pods)
kubectl audit pods -n ns-prod

# Pods: all namespaces (adds a NAMESPACE column; same idea as kubectl get pods -A)
kubectl audit pods -A
kubectl audit po --all-namespaces

# Pods: label filter with all namespaces
kubectl audit pods -A -l app=web

# PVC / PV
kubectl audit pvc -A
kubectl audit pv

# Jobs and CronJobs
kubectl audit job -A
kubectl audit cj -A

# Services: selectors with no matching Pods (see also Service audit demo below)
kubectl audit service
kubectl audit svc -A
kubectl audit service -n my-namespace -l app=myapp

# Deployments: spec.replicas 0 (scaled to zero)
kubectl audit deploy
kubectl audit deploy -A

# Portable demo YAML per audit: see examples/README.md and examples/audit-*/

```

### Example manifests by audit

Each subfolder under [`examples/`](examples/README.md) has a **`demo.yaml`** plus a **README** with apply / audit / delete steps.

| Folder | `kubectl audit` target |
| ------ | ------------------------ |
| [`examples/audit-pods/`](examples/audit-pods/README.md) | `pods` |
| [`examples/audit-containers/`](examples/audit-containers/README.md) | `containers` ([`demo.yaml`](examples/audit-containers/demo.yaml): failing **init**; [audit-pods](examples/audit-pods/README.md) for bad **image**) |
| [`examples/audit-job/`](examples/audit-job/README.md) | `jobs` |
| [`examples/audit-cronjob/`](examples/audit-cronjob/README.md) | `cronjobs` |
| [`examples/audit-svc/`](examples/audit-svc/README.md) | `service` |
| [`examples/audit-deploy/`](examples/audit-deploy/README.md) | `deploy` |
| [`examples/audit-pvc/`](examples/audit-pvc/README.md) | `pvc` |
| [`examples/audit-pv/`](examples/audit-pv/README.md) | `pv` |

**Nodes** are not included (cluster-specific). There is no `audit-nodes` example folder.

### Sample output

**Pods — single namespace**

```bash
$ kubectl audit pods # kubectl audit pods -n ns-prod
-------------------------------------------------------
Pod Audit summary: total = 9 benign = 3 attention = 6
-------------------------------------------------------
Pod that requires attention
NAME                                            READY   STATUS                  RESTARTS   AGE
workload-a-dep-54b6948c9c-pqx12                 0/1     ImagePullBackOff        0          8h
workload-a-dep-785b496f5d-rst34                 0/1     ContainerCreating       0          11h
svc-b-dep-6c87c74674-uvw56                      2/3     ErrImagePull            0          3h5m
svc-b-dep-9f6d7b5cc-xyz99                       0/3     ContainerStatusUnknown  0          12h
sidecar-c-dep-9c5c948dd-abccd                   0/1     CrashLoopBackOff        0          18h
sidecar-c-dep-c45d7c6c5-efghi                   0/1     InvalidImageName        0          8h
```

**Pods — all namespaces**

```bash
$ kubectl audit pods -A
-------------------------------------------------------
Pod Audit summary: total = 120 benign = 114 attention = 6
-------------------------------------------------------
Pod that requires attention
NAMESPACE      NAME                                            READY   STATUS             RESTARTS   AGE
ns-team-a      workload-dep-5f7677d8c-pqx12                    0/1     ImagePullBackOff   0          8h
ns-team-a      workload-dep-6a8b9c0d-rst34                     0/1     ImagePullBackOff   0          11h
ns-team-b      indexer-dep-7c8d9e0f-uvw56                      0/1     ImagePullBackOff   0          3h55m
ns-shared      sidecar-dep-8b9c0d1e-xyz78                      0/1     ErrImagePull       0          45m
ns-shared      batch-harness-dep-9c0d1e2f-ab901                1/1     Running            6          4h
ns-monitoring  obs-collector-dep-0d1e2f3g-cd234                1/1     Running            18         2d
```

*(The `Running` rows are listed when `RestartCount` reaches the attention threshold; other rows are not in a healthy steady state.)*

**Containers** (illustrative columns; default omits image / ports / pull policy unless `-o wide`)

```bash
$ kubectl audit containers -n demo
-------------------------------------------------------
Container Audit summary: total = 12 benign = 9 attention = 3
-------------------------------------------------------
Container that requires attention
POD                     NAME                    READY   STATUS             RESTARTS   AGE     TYPE
workload-dep-abc-xyz    sidecar                 0       ImagePullBackOff   0          -       container
```

**Nodes**

```bash
$ kubectl audit nodes
-------------------------------------------------------
Node Audit summary: total = 66 benign = 46 attention = 20
-------------------------------------------------------
Node that requires attention
NAME            STATUS                        ROLES    AGE     VERSION
default-0       Ready,SchedulingDisabled      <none>   459d    v1.30.3
default-1       Ready,SchedulingDisabled      <none>   35d     v1.30.3
default-2       Ready,SchedulingDisabled      <none>   35d     v1.30.3
worker-0        NotReady,SchedulingDisabled   <none>   2d21h   v1.33.3
worker-1        NotReady,SchedulingDisabled   <none>   23h     v1.33.3
worker-2        NotReady,SchedulingDisabled   <none>   2d19h   v1.33.3
worker-3        NotReady                      <none>   24h     v1.33.3
worker-4        NotReady,SchedulingDisabled   <none>   45h     v1.33.3
...
...
```

**Persistent volumes**

```bash
$ kubectl audit pv
-------------------------------------------------------
PersistentVolume Audit summary: total = 12 benign = 11 attention = 1
-------------------------------------------------------
PersistentVolume that requires attention
NAME              CAPACITY   ACCESS MODES   STATUS      CLAIM
pv-archive-001    500Gi      RWO            Released    demo-ns/pvc-old-claim
```

**Persistent volume claims**

```bash
$ kubectl audit pvc -A
-------------------------------------------------------
PersistentVolumeClaim Audit summary: total = 45 benign = 43 attention = 2
-------------------------------------------------------
PersistentVolumeClaim that requires attention
NAMESPACE    NAME              STATUS    VOLUME
app-demo     logs-claim-01     Pending
data-demo    backup-claim-02   Lost
```

**Jobs**

```bash
$ kubectl audit jobs -A
-------------------------------------------------------
Job Audit summary: total = 28 benign = 26 attention = 2
-------------------------------------------------------
Job that requires attention
NAMESPACE    NAME               COMPLETIONS   DURATION   AGE
batch-demo   daily-import       0/1           5m         5m
batch-demo   retry-migrate      0/1           1h         1h
```

**CronJobs**

```bash
$ kubectl audit cronjobs -A
-------------------------------------------------------
CronJob Audit summary: total = 15 benign = 13 attention = 2
-------------------------------------------------------
CronJob that requires attention
NAMESPACE   NAME               SCHEDULE      SUSPEND   ACTIVE
ops-demo    pause-backup       0 2 * * *     True      0
ops-demo    hold-reports       15 * * * *    True      0
```

For machine-oriented `-o` (including **`containers`** `json` / `yaml` / `name` and **`kubectl get`–style** output for the other subcommands), the audit summary line is written to **stderr** so you can pipe **stdout** to `jq` or other tools unchanged.

## Development

**Prerequisites:** Go 1.21+, `make`, and a working cluster context if you want to run the plugin end to end.

```bash
git clone https://github.com/codenio/kubectl-audit.git
cd kubectl-audit
make bin          # writes bin/audit
make test         # tests + coverage profile
make fmt && make vet
make install    # installs the plugin to ~/.krew/kubectl-audit for ad-hoc/cluster testing
```

Run without installing (same flags as under `kubectl audit`):

```bash
go run ./cmd/plugin --help
go run ./cmd/plugin pods
go run ./cmd/plugin containers --help
```

**Repository layout**

- `cmd/plugin/` — entrypoint and CLI (`cobra`, config flags, printing).
- `pkg/plugin/` — audit logic, container list/table data (`containers.go`), and server-side table handling.
- `deploy/krew/plugin.yaml` — Krew manifest template for releases.

To bump pinned Kubernetes dependencies, use the `kubernetes-deps` target in the `Makefile`.

## Contributing

Issues and pull requests are welcome: [github.com/codenio/kubectl-audit/issues](https://github.com/codenio/kubectl-audit/issues).

Before you open a PR:

- `make test` passes.
- `make bin` succeeds and `make fmt` / `make vet` are clean (or run `make bin`, which runs `fmt` and `vet` first).

Validate changes against a cluster with `make install` and `kubectl audit …` as needed.

## Acknowledgments

This plugin was created using [replicatedhq/krew-plugin-template](https://github.com/replicatedhq/krew-plugin-template).