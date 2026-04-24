# Example manifests

Each folder is self-contained: apply **`demo.yaml`**, run the matching **`kubectl audit …`** command, then delete the same file.

| Folder | Command |
| ------ | ------- |
| [audit-pods](audit-pods/README.md) | `kubectl audit pods` |
| [audit-containers](audit-containers/README.md) | `kubectl audit containers` (failing **init** container + healthy Pod; see [audit-pods](audit-pods/README.md) for bad **image** on the main container) |
| [audit-job](audit-job/README.md) | `kubectl audit jobs` |
| [audit-cronjob](audit-cronjob/README.md) | `kubectl audit cronjobs` |
| [audit-svc](audit-svc/README.md) | `kubectl audit service` |
| [audit-deploy](audit-deploy/README.md) | `kubectl audit deploy` |
| [audit-pvc](audit-pvc/README.md) | `kubectl audit pvc` |
| [audit-pv](audit-pv/README.md) | `kubectl audit pv` |

There is **no** `audit-nodes` sample (not portable across clusters). See the main [README](../README.md) for `kubectl audit nodes`.
