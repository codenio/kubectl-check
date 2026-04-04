# kubectl-audit

`[kubectl-audit](https://github.com/codenio/kubectl-audit)` is a `[kubectl` plugin]([https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/](https://kubernetes.io/docs/tasks/extend-kubectl/kubectl-plugins/)) that lists Kubernetes resources failing common health checks: unhealthy or crash-prone pods (including high container restarts), unhealthy nodes, unbound volumes, failed jobs, and suspended cron jobs. Output uses the same printers as `kubectl get` (default table, `-o wide`, JSON, YAML, custom columns, Go templates, and more).

## Contents

- [Install](#install)
- [Usage](#usage)
- [Resources and filters](#resources-and-filters)
- [Output formats](#output-formats)
- [Examples](#examples)
  - [Sample output](#sample-output)
- [Development](#development)
- [Contributing](#contributing)
- [Acknowledgments](#acknowledgments)

## Install

### Krew (codenio custom index)

Published through [codenio-krew-index](https://github.com/codenio/codenio-krew-index):

```bash
kubectl krew index add codenio https://github.com/codenio/codenio-krew-index.git
kubectl krew install codenio/audit
```

Upgrade when the index is updated:

```bash
kubectl krew upgrade codenio/audit
```

If the local index name `codenio` is already taken, pick another name for `kubectl krew index add` and use that same prefix in `install` / `upgrade` (for example `codenio-krew/audit`). Maintainer notes for bumping versions live in the index repository.

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

Every invocation needs a **resource** argument (see [Resources and filters](#resources-and-filters)):

```bash
kubectl audit pods
kubectl audit --help
```

Standard `kubectl` config applies: current context, `KUBECONFIG`, `-n` / `--namespace`, `--context`, and so on.

## Resources and filters


| Resource   | Aliases                                                   | What is listed                                                                                                                                                                                                                                                                        |
| ---------- | --------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `pods`     | `pod`, `po`                                               | Pods that need attention: phase is not `Running`, any regular container is not `Ready`, **or** any regular or init container has `RestartCount` ≥ **5** (threshold is fixed in code). `Succeeded` / `Completed` job pods are included because they are not in a running steady state. |
| `nodes`    | `node`, `no`                                              | Nodes that are `NotReady` or have `SchedulingDisabled`.                                                                                                                                                                                                                               |
| `pvc`      | `pvcs`, `persistentvolumeclaim`, `persistentvolumeclaims` | PVCs not in `Bound` phase.                                                                                                                                                                                                                                                            |
| `pv`       | `pvs`, `persistentvolume`, `persistentvolumes`            | PVs not in `Bound` phase.                                                                                                                                                                                                                                                             |
| `jobs`     | `job`                                                     | Failed jobs (including backoff / deadline failures).                                                                                                                                                                                                                                  |
| `cronjobs` | `cronjob`, `cj`                                           | Suspended cron jobs.                                                                                                                                                                                                                                                                  |


**Common flags**

- **All namespaces:** `-A` or `--all-namespaces` for namespaced resources (`pods`, `pvc`, `jobs`, `cronjobs`).
- **Labels:** `-l` / `--selector` (same semantics as `kubectl get`).

There are no separate pod subcommands or `--pending` / `--failed` style switches: one `kubectl audit pods` run applies all pod rules above.

Further notes live in [doc/USAGE.md](doc/USAGE.md).

## Output formats

`[kubectl get](https://kubernetes.io/docs/reference/kubectl/generated/kubectl_get/)`-style `-o` flags are supported, for example:

```bash
kubectl audit pods -o wide
kubectl audit nodes -o json
kubectl audit pvc -o yaml
kubectl audit jobs -o custom-columns=NAME:.metadata.name
```

## Examples

```bash
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

```

### Sample output

For default and wide output, stdout matches this layout: two `-------------------------------------------------------` lines (55 hyphens, same as the plugin), the `**{Kind} Audit summary: total = … benign = … attention = …**` line, `**{Kind} that requires attention**`, then the same table as `kubectl get` (including **RESTARTS** for pods). On a **terminal (TTY)**, there is no extra blank line between the summary line and the second rule; when stdout is **piped or redirected**, the plugin inserts one blank line after the summary line before the second rule.

- **total** — resources in scope (full list before the audit filter).
- **benign** — resources that pass the audit’s “OK” bar for that kind (for **pods**: `Running`, every regular container `Ready`, and no regular or init container has `RestartCount` of 5 or more).
- **attention** — rows in the filtered result (same as the table).

If the table is empty, **stderr** explains either that nothing of that kind exists in scope or that nothing requires attention; **stdout** still prints the summary block.

Names and namespaces below are **illustrative and masked**; spacing is aligned like typical `kubectl get` output.

**Pods — single namespace**

```bash
$ kubectl audit pods # kubectl audit pods -n ns-prod
-------------------------------------------------------
Pod Audit summary: total = 9 benign = 3 attention = 6
-------------------------------------------------------
Pod that requires attention
NAME                                            READY   STATUS             RESTARTS   AGE
workload-a-dep-54b6948c9c-pqx12                 0/1     ImagePullBackOff   0          8h
workload-a-dep-785b496f5d-rst34                 0/1     ImagePullBackOff   0          11h
svc-b-dep-6c87c74674-uvw56                      0/1     ImagePullBackOff   0          3h55m
svc-b-dep-9f6d7b5cc-xyz99                       0/1     ImagePullBackOff   0          12h
sidecar-c-dep-9c5c948dd-abccd                   0/1     ImagePullBackOff   0          18h
sidecar-c-dep-c45d7c6c5-efghi                   0/1     ImagePullBackOff   0          8h
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

**Nodes**

```bash
$ kubectl audit nodes
-------------------------------------------------------
Node Audit summary: total = 53 benign = 51 attention = 2
-------------------------------------------------------
Node that requires attention
NAME               STATUS     ROLES    AGE
node-worker-b02    NotReady   worker   40d
node-worker-d04    NotReady   worker   30d
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

``bash
$ kubectl audit cronjobs -A
-------------------------------------------------------
CronJob Audit summary: total = 15 benign = 13 attention = 2
-------------------------------------------------------
CronJob that requires attention
NAMESPACE   NAME               SCHEDULE      SUSPEND   ACTIVE
ops-demo    pause-backup       0 2 * * *     True      0
ops-demo    hold-reports       15 * * * *    True      0
```

For `-o json`, `-o yaml`, and other machine-oriented formats, the summary line is written to **stderr** so you can pipe **stdout** to `jq` or other tools unchanged.

## Development

**Prerequisites:** Go 1.21+, `make`, and a working cluster context if you want to run the plugin end to end.

```bash
git clone https://github.com/codenio/kubectl-audit.git
cd kubectl-audit
make bin          # writes bin/audit
make test         # tests + coverage profile
make fmt && make vet
```

Run without installing (same flags as under `kubectl audit`):

```bash
go run ./cmd/plugin --help
go run ./cmd/plugin pods
```

**Repository layout**

- `cmd/plugin/` — entrypoint and CLI (`cobra`, config flags, printing).
- `pkg/plugin/` — audit logic and server-side table handling.
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