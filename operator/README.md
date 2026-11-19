# mip-notebook-operator

Builds JupyterHub notebook pods from `Notebook` / `NotebookProfile` resources so
the hub needs no permission on pods or PVCs. With `--mode=spawn-api` the same
image serves the hubs' HTTPS spawn API, so they need no Kubernetes credential
at all. Design, RBAC and rollout:
[documentation/notebook-operator.md](../documentation/notebook-operator.md).

Scaffolded with Operator SDK (`go/v4` plugin).

```bash
make test                    # envtest controller and spawn API tests
make manifests generate      # after editing api/ or RBAC markers; commit the output
docker build -t hbpmip/mip-notebook-operator:<tag> .
```

CRDs: `config/crd/bases/`. Operator RBAC: `config/rbac/role.yaml` (generated).
`testdata/` holds the spawn API RBAC and the policy on the token Secrets that the
tests apply: hand-maintained copies of mip-infra's
`base/mip-infrastructure/notebook-operator/rbac.yaml`, not deployed from here.
The deployments live in `mip-infra`. The controller (`--mode=controller`,
the default) needs `WATCH_NAMESPACES` (comma separated) and leader election
(`--leader-elect`). The spawn API (`--mode=spawn-api`) needs neither; it takes
`--spawn-api-bind-address`, `--tls-cert-dir`, `--health-probe-bind-address`,
and optionally `--metrics-bind-address` and `--spawn-api-rate-limit` /
`--spawn-api-rate-burst`.
