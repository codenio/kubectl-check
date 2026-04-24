# Example: `kubectl audit deploy`

Manifests: [`demo.yaml`](demo.yaml) — namespace **`kubectl-audit-deploy-demo`**, label **`demo=kubectl-audit-deploy`**.

- **`kubectl-audit-demo-idle`** — `spec.replicas: 0` → listed by the audit.
- **`kubectl-audit-demo-running`** — `replicas: 1` → not listed.

## Try it

```bash
kubectl apply -f examples/audit-deploy/demo.yaml
kubectl wait --for=condition=available deployment/kubectl-audit-demo-running -n kubectl-audit-deploy-demo --timeout=120s
kubectl audit deploy -n kubectl-audit-deploy-demo -l demo=kubectl-audit-deploy
kubectl delete -f examples/audit-deploy/demo.yaml
```
