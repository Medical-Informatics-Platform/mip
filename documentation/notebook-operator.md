# Notebook operator

JupyterHub no longer creates pods or PVCs. It creates a `Notebook` custom
resource; the MIP notebook operator (`operator/`, Operator SDK `go/v4`) turns it
into a pod and a PVC from a `NotebookProfile` the hub cannot edit.

## Why

With KubeSpawner the hub needed a Role with `pods`/`persistentvolumeclaims`
create/delete in every `federation-*` namespace. When Argo CD creates that
Role, RBAC escalation prevention forces the Argo application controller to hold
`pods: create` (and RBAC write) cluster-wide. A compromised hub could run any image in a
namespace that also holds the platform databases and Exareme workers.

After the switch:

| Component | Can create pods | How |
|---|---|---|
| JupyterHub, `operator` mode | no | `notebooks` create/get/list/watch/delete, `secrets` create (Role applied out-of-band by an admin) |
| JupyterHub, `operator-api` mode | no | no Kubernetes credential at all; calls the spawn API |
| Spawn API | no | `notebooks` get/create/delete, `secrets` create per federation, `tokenreviews` create |
| Argo CD controller | no | no RBAC write either; only syncs `NotebookProfile` |
| Notebook operator | yes, only in namespaces an admin bound | ClusterRole + per-namespace RoleBinding; no access to Secrets |

`operator-api` is the target: a compromised hub then holds nothing the
Kubernetes API server accepts.

## Resources

API group `notebooks.mip.ebrains.eu/v1alpha1`, both namespaced.

### NotebookProfile

Shipped by the chart in this repository (`deployment/kubernetes`), synced by Argo CD. The hub has no access.

```yaml
spec:
  image: hbpmip/mip-jupyter:0.0.1_candidate
  imagePullPolicy: IfNotPresent
  resources: {requests: {cpu: 500m, memory: 1G}, limits: {cpu: "1", memory: 4G}}
  storage:
    storageClassName: ceph-corbo-cephfs
    size: 2Gi
    mountPath: /home/jovyan
    fixOwnership: true    # root init container chowns the volume to 1000:100 (provisioners that ignore fsGroup)
  nodeSelector: {}
  env:                    # static, non-secret
    - {name: PLATFORM_BACKEND_URL, value: http://platform-backend-service:8080/services}
```

The operator, not the profile, sets the security context: `runAsUser: 1000`,
`runAsGroup: 100`, `fsGroup: 100`, `runAsNonRoot`, seccomp `RuntimeDefault`,
all capabilities dropped, no privilege escalation, no service account token.
The notebook listens on port 8888.

### Notebook

Created per server by the hub, or by the spawn API on its behalf.

```yaml
metadata: {name: jupyter-{user_server}}  # KubeSpawner's pod name ("safe" slugs)
spec:
  profile: default
  user: <hub user name>
  serverName: ""
  env: {JUPYTERHUB_SERVICE_PREFIX: /notebook/user/<user>/, ...}
status:
  phase: Pending | Running | Failed
  url: http://<pod ip>:8888
  message: <latest reason>
```

CRD validation (CEL, enforced by the API server):

- `spec` is immutable; a restart is delete + create.
- `env` keys match `^(JUPYTERHUB|JPY)_[A-Z0-9_]+$` and exclude
  `JUPYTERHUB_API_TOKEN` / `JPY_API_TOKEN`.
- Names start with `jupyter-` and are at most 57 characters.

The hub (or, in `operator-api` mode, the spawn API) puts the hub API token in
Secret `<notebook name>-<notebook uid>-token` (key `token`, owned by the
`Notebook`). The API server assigns the UID, so no
Secret that existed before the `Notebook` is ever mounted: not
`keycloak-credentials`, and not a token left by an earlier `Notebook` of the
same name. The hub can create Secrets but not overwrite them. The
ValidatingAdmissionPolicy `mip-jupyterhub-secrets` (mip-infra, test copy in
`operator/testdata/token-secrets-policy.yaml`)
admits only Opaque Secrets named and owned that way from the hub and from the
spawn API (both, so hubs can move one at a time), so neither can
create a `kubernetes.io/service-account-token` Secret, which the API server
would fill with another ServiceAccount's token, for a pod to mount. The
operator never reads Secrets.

## Operator

One Deployment in `mip-notebooks-system`, leader election on, watching only the
namespaces in `WATCH_NAMESPACES` (comma separated). Adding a federation means
adding it to that list and binding the ClusterRole there.

Shared examples (optional): with `EXAMPLES_CLAIM` set (mip-infra uses
`mip-examples`), every new notebook pod mounts that claim read-only at
`work/.mip-examples` under the profile's `storage.mountPath`, where the
single-user entrypoint links `examples/` to it (every pod gets `MIP_WORK_DIR` =
`<mountPath>/work`, so the entrypoint looks there). The mount is added only when the
claim in the notebook's namespace is `Bound` (and not being deleted) and
the profile sets `fixOwnership: true` (its init container creates `work/`, so the
runtime does not create it as root for the mount point). A missing or unbound
claim is skipped and the notebook starts without examples; other API
errors retry the reconcile. A claim pruned between that check and pod creation
leaves the pod `Pending` until the notebook is recreated. The decision is in the
"created notebook pod" log line. Rollback: unset `EXAMPLES_CLAIM`, and new
notebooks start without the mount; running ones keep it until they restart.

Reconcile of a `Notebook`:

1. Deleted: nothing to do. Pod and token Secret are owned by the `Notebook` and
   garbage collected.
2. No pod yet: load the profile. Missing: `Failed`, retried every 30s. A running
   pod does not need its profile, so deleting one does not stop servers.
3. Ensure PVC `claim-{user_server}`, KubeSpawner's name for the user's default
   server, so homes KubeSpawner created are reused. Never owned, updated or
   deleted, so home directories survive restarts.
4. Ensure pod (bare pod, owned by the `Notebook`). A pod that ends is not
   recreated; the hub sees `Failed` and the user starts again. Until the hub
   has created the token Secret the kubelet holds the pod in
   `CreateContainerConfigError`, reported as the message. If the API server
   rejects the PVC or pod (quota, Pod Security), `Pending` with that error,
   retried with backoff.
5. Status from the pod: `Running` + `url` when Ready, `Pending` with the waiting
   reason (`ImagePullBackOff`, `Unschedulable`, ...), `Failed` with the
   termination message. A pod of that name the `Notebook` does not own, such as
   one KubeSpawner left, is `Failed`: delete it before starting again.

Profile edits apply from the next start; running pods are not touched.

Operator RBAC (ClusterRole `mip-notebook-operator`, generated in
`operator/config/rbac/role.yaml`, bound per namespace): `pods` and
`persistentvolumeclaims` get/list/watch/create, `notebooks` and
`notebookprofiles` get/list/watch, `notebooks/status` get/update/patch. No
Secrets, no delete: garbage collection removes pods.

## Spawn API

The same image with `--mode=spawn-api`: a separate Deployment in
`mip-notebooks-system`, ServiceAccount `mip-notebook-spawn-api`, 2 replicas,
no leader election (stateless), no informer cache (a direct client; RBAC
limits it to the namespaces it is bound in). Served at
`https://mip-notebook-spawn-api.mip-notebooks-system.svc.cluster.local`
(Service 443 to container 8443).

```text
--mode=spawn-api
--spawn-api-bind-address=:8443
--tls-cert-dir=/etc/mip-notebook-spawn-api/tls   # tls.crt, tls.key; reloaded on change
--health-probe-bind-address=:8081                # /healthz, /readyz
--metrics-bind-address=0                         # off by default
--spawn-api-rate-limit=5 --spawn-api-rate-burst=20   # per namespace
```

TLS 1.2 minimum. The certificate (cert-manager in mip-infra) must name the
Service DNS name above: the hub verifies it against its CA file.

### Authentication and authorization

`Authorization: Bearer <token>`, checked with a TokenReview for audience
`mip-notebook-operator`. The request passes only when the review says
`authenticated` and lists that audience in `status.audiences`. Every request
makes its own TokenReview, uncached: a pod-bound token stops working as soon as
its hub pod is deleted. Hub traffic (one PUT per spawn, then status polls) keeps
that load small.

The user must be exactly `system:serviceaccount:{ns}:jupyterhub`, where `{ns}`
is the namespace in the path and matches `^federation-[a-z0-9-]+$`. Anything
else is 403. Then a token bucket per namespace (429 with `Retry-After`).
Each denial is logged with namespace, remote address, reason and user name;
tokens and bodies are never logged. Metrics: `mip_spawn_api_requests_total`
(method, code) and `mip_spawn_api_denied_total` (reason).

### Endpoints

JSON; errors are `{"error": "<message>"}`. 401 unauthenticated, 403
forbidden, 429 rate limited, 503 when the TokenReview itself fails.

| Request | Result |
|---|---|
| `PUT /v1/namespaces/{ns}/servers/{name}` | 201 created; 200 same spec again; 409 other spec or terminating; 400 invalid; 413 body over 64 KiB |
| `GET /v1/namespaces/{ns}/servers/{name}` | 200 `{"phase", "url", "message", "deleting"}`; 404 gone |
| `DELETE /v1/namespaces/{ns}/servers/{name}` | 202 while the `Notebook` exists (foreground delete); 404 gone |

PUT body: `{"profile", "user", "serverName", "env", "apiToken"}`, unknown
fields rejected. `name` and `env` follow the CRD rules (`jupyter-*`, at most
57 characters, DNS label; `JUPYTERHUB_*`/`JPY_*` keys, no token keys);
`apiToken` is required. The spawn API creates the `Notebook` with annotation
`notebooks.mip.ebrains.eu/requested-by: <hub account>` (the API server's
audit log now shows only the spawn API), then the token Secret (Opaque, one
controller owner reference to the `Notebook`). If the Secret fails it deletes
the `Notebook` and answers 500.

The spawn API cannot read Secrets, so a PUT for an existing `Notebook` with the
same spec creates the Secret to find out whether it exists: an `AlreadyExists`
means it does, a success completes a request that stopped between the two.
It cannot tell whether an existing Secret holds the same `apiToken`; the hub
always deletes and waits for 404 before a PUT, so that only arises for a retry
of the same start.

### Why a separate account

The controller can create pods in every federation, which means it can read
every Secret there by mounting it. The spawn API faces the hubs over the
network. Kept apart:

- the controller has no listener the hubs can reach, and never gets
  `secrets: create`;
- the spawn API has no pods, PVCs, Secret reads, list or watch. Compromised, it
  can start notebooks from the profiles the hubs already use and write token
  Secrets the admission policy admits, in the federations it is bound in.

RBAC, deployed from mip-infra
`base/mip-infrastructure/notebook-operator/rbac.yaml`. The copy the tests apply,
`operator/testdata/spawn-api-role.yaml`, is hand-maintained (not kubebuilder
markers, which would merge into `manager-role`); change both together:

- ClusterRole `mip-notebook-spawn-api`, bound per federation namespace:
  `notebooks` get/create/delete, `secrets` create.
- ClusterRole `mip-notebook-spawn-api-tokenreview`, bound cluster-wide:
  `tokenreviews` create.

## Hub spawner

`JUPYTERHUB_SPAWNER` selects one (default `kubespawner` until the operator is
installed):

- `operator`: mip-jupyter `docker/hub/notebook_spawner.py`, talks to the Kubernetes API.
- `operator-api`: mip-jupyter `docker/hub/notebook_api_spawner.py`, talks to the spawn API
  with tornado's `AsyncHTTPClient`. Imports no Kubernetes client.

Both:

- `start()`: take `JUPYTERHUB_API_TOKEN` / `JPY_API_TOKEN` out of
  `get_env()`, create the `Notebook` (replacing a leftover one), create the
  token Secret `<name>-<uid>-token` owned by it, wait for `Running`, return
  `status.url`. On timeout delete the `Notebook`.
- `progress()`: yields `status.message` changes.
- `poll()`: `None` while `Pending`/`Running`, `1` when `Failed` or gone.
- `stop()`: delete the `Notebook` (foreground) and wait until it is gone.

`operator-api` maps these onto the endpoints above: `start()` is DELETE and
wait for 404 (60 s), PUT, then GET every second until `Running` with a `url`
or `Failed` (then DELETE and wait). It retries 429 (after `Retry-After`), 502,
503, 504 and no response, up to 6 attempts. Environment (chart in this repository):

| Variable | Value |
|---|---|
| `NOTEBOOK_API_URL` | `https://mip-notebook-spawn-api.mip-notebooks-system.svc.cluster.local` |
| `NOTEBOOK_API_TOKEN_FILE` | `/var/run/secrets/mip-notebook-operator/token`: projected, audience `mip-notebook-operator`, 3600 s; read on every request |
| `NOTEBOOK_API_CA_FILE` | `/var/run/secrets/mip-notebook-operator/ca.crt`: passed as `ca_certs`, loaded per connection |
| `POD_NAMESPACE` | downward API |

The hub pod sets `automountServiceAccountToken: false`.

`MIP_TOKEN` is no longer put in the pod env (the `env` rule rejects it). The
`mip` client fetches the first token from the hub `/api/platform-token`
endpoint, as it already did for refreshes.

## Rollout

1. `mip`: operator image; `mip-jupyter`: hub image with both spawners, client change.
2. `mip-infra`: install CRDs + operator, the hub Secret policy (Kubernetes 1.30+),
   out-of-band operator and hub RBAC, whitelist `NotebookProfile` in the federation AppProject, Argo
   controller gets `notebookprofiles` write.
3. `mip` chart: `jupyterhub.spawner: operator` renders the profile;
   `jupyterhub.rbac.create: false` leaves the hub Role to the admin.
4. `mip-infra`: drop `pods: create` and RBAC write from the Argo
   controller.

In `operator` mode the operator and hub images must agree on the token Secret
name and the CRD. Upgrade the CRD and operator first, then the hub, with no
server running in operator mode in between.

Moving to `operator-api`, per federation:

1. `mip-infra`: the spawn API Deployment, Service and certificate; the
   updated Secret policy (admits both accounts); RoleBinding to
   `mip-notebook-spawn-api` in the federation, ClusterRoleBinding to
   `mip-notebook-spawn-api-tokenreview` once.
2. `mip` chart: `operator-api` mode renders the env above, the
   projected token and CA volume, and `automountServiceAccountToken: false`.
   Running servers survive: both modes use the same `Notebook` names.
3. Drop the hub Role and RoleBinding. Once every federation has moved, drop
   the `jupyterhub` match from the Secret policy.

## Contracts with other repositories

The operator is built here, the hub and single-user images in mip-jupyter, and
the cluster installation is in mip-infra. A change to any of these needs both sides:

| Between | Contract |
|---|---|
| Hub spawner, spawn API | `/v1` endpoints, audience `mip-notebook-operator`, name rule (`jupyter-*`, at most 57 characters), `env` key rule |
| Hub `operator` spawner, operator | `Notebook` CRD, token Secret name `<name>-<uid>-token`. Goes with that spawner once every federation runs `operator-api` |
| Single-user image, operator | Port 8888, uid `1000:100`, `MIP_WORK_DIR`, `work/.mip-examples` |
| mip-infra, operator | Flags and env above, ClusterRoles, policy `mip-jupyterhub-secrets`, CRDs from `operator/config/crd/bases` at the release tag |

## Testing

- Operator: `cd operator && make test` (envtest: CRD validation rules, hub
  Secret policy, profile missing, pod build, status mapping, KubeSpawner PVC
  names, shared examples mount).
- Spawn API: `cd operator && make test` also runs `internal/spawnapi`
  (envtest: TokenReview audiences, authorization, PUT/GET/DELETE semantics,
  Secret rollback, rate limit, TLS reload, and a spawn as the spawn API
  account with only its RBAC through the Secret policy, mounted by the
  controller).
- Spawners: mip-jupyter `docker/hub/test_notebook_spawner.py` and
  `test_notebook_api_spawner.py`, run inside the hub image (CI there does this
  after building it).
- Staging: spawn, stop, restart the hub with a server running, and confirm
  `kubectl auth can-i create pods --as=system:serviceaccount:<ns>:jupyterhub`
  returns `no`. In `operator-api` mode also confirm the hub pod has no
  `/var/run/secrets/kubernetes.io/serviceaccount`, and that its projected
  token is refused by the API server (401).
