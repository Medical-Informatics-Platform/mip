# Kubernetes Deployment

## Infrastructure Setups
MIP now supports two Kubernetes infrastructure options:

1. **VM-based / microk8s clusters** – the remainder of this document (starting in the Requirements section) walks through preparing Ubuntu virtual machines and installing the stack on top of microk8s.
2. **Managed clusters** – for cloud-managed Kubernetes (AKS/EKS/GKE, etc.) follow the [mip-infra getting started guide](https://github.com/Medical-Informatics-Platform/mip-infra?tab=readme-ov-file#-getting-started) to provision the cluster and its base services. Once the cluster is available, return here for component configuration details as needed.

Choose between these modes via the Helm values: set `cluster.managed: true` (managed) or `false` (microk8s/VM). The chart now keeps the same workload shape in both modes and uses the flag only to select the storage class and, for microk8s, bootstrap the local `StorageClass`.

## Requirements
### Hardware
#### Master node
* 60 GB HDD
* 16 GB RAM
* 4 CPU Cores

#### Worker node
* 40 GB HDD
* 8 GB RAM
* 2 CPU Cores

### Software
From now on, most of our deployments will be done with Ubuntu Server 22.04, but as we run all the MIP containers on top of microk8s (as the Kubernetes distribution), it may be possible (never tested) to run it on other operating systems, including Mac OS, and Windows.

## Components:
Now, with the Kubernetes (K8s) deployment, we have 2 main component packs, that need to be deployed, which come as Helm charts:

### The analysis engine: Exaflow

Deploy the [Exaflow 1.3.0 chart](https://github.com/madgik/exaflow/tree/1.3.0/kubernetes)
for the native federated pipeline:

- Controller: accepts analyses and coordinates execution.
- Local workers: load each site's datasets and execute local computations.
- Aggregation server: combines partial vectors for algorithms and preprocessing
  steps that require aggregation. Keep `aggregation_server.enabled: true` for
  the full native algorithm portfolio.

### The web and notebook stack

- **platform-ui**: analysis studio, result visualizations, and notebook entry.
- **platform-backend**: authenticated analysis API and study management.
- **PostgreSQL**: stores platform analyses, studies, and other application state.
- **JupyterHub and single-user JupyterLab**: notebook sessions and workspaces,
  using images from **mip-jupyter**. Hub state and user homes use separate PVCs.
- **External Keycloak**: provides authentication; this chart wires an existing
  realm and client credentials rather than deploying Keycloak.

Component versions are listed in [MIP Components](../../documentation/Components.md).
The database bootstrap script is vendored at
`files/platform-backend-db-init.sh` and mounted through a ConfigMap. Keep it
aligned with the backend release when updating the chart.

## Dataset placement

Prepare CSV datasets and `CDEsMetadata.json` according to the
[dataset onboarding guide](../../documentation/MIP_Data_management_documentation.md).
For unmanaged Exaflow clusters, the chart mounts
`<storage.hostPath.db.localworker>/csvs` from each worker node into `/opt/csvs`.
Each data-model directory belongs under that `csvs` directory. With the example
below, this is `/data/<MIP_INSTANCE_OR_FEDERATION_NAME>/localworker/csvs`.
Prepare the global worker's `csvs` directory on the master node as well.

For managed Exaflow clusters, populate the CSV PVCs provisioned by the Exaflow
chart. Data placement and backups remain the responsibility of each data site.
The worker DuckDB database is rebuilt from these inputs on startup.

Researcher filters and transformations are configured per analysis in the
Experiment Studio or notebook pipelines. They do not replace dataset onboarding
or alter the source CSV files.

## Configuration
Prior to deploying it (on a microk8s K8s cluster of one or more nodes), there are a few adjustments to make in `values.yaml`. Each top-level section controls a part of the stack:

* `cluster`: storage classes and whether the chart should use the managed storage class or the microk8s local storage class.
* `global`: shared public hostname used by the ingress and backend redirects.
* `platform-ui`, `platform-backend`, `platformBackendDatabase`: container images and component specific options (including the shared ingress/tls settings and PVC sizes).
* `platform-ui.notebook`: enables the `/notebook` iframe route in platform-ui and wires nginx to the in-cluster JupyterHub service.
* `jupyterhub`: JupyterHub and single-user Jupyter images, hub resources, ingress, storage, crypt key, and notebook resource requests/limits. Rendered only when `platform-ui.notebook.enabled` is true, which also requires `keycloak.enabled`. Notebook traffic for the portal iframe should use platform-ui's `/notebook/` proxy; keep `jupyterhub.ingress.enabled: false` unless you need a separate admin-only host.
  Notebook examples are mounted only in operator mode on MIP infra clusters (mip-infra
  `common/examples`). KubeSpawner installs start without them.
* `keycloak`: toggles the connection parameters to the external Keycloak instance (`enabled`, `host`, `protocol`, `realm`), and `cliClientId` for using MIP from users' own computers (below).
* `rabbitmq`: single-node RabbitMQ 4.4.1 broker for Exaflow AMQP analyses. Create the `exaflow-rabbitmq` Secret out of band before deploying. The broker requests 512Mi and is limited to 2Gi.

Copy `values.yaml` to a new file (for example `my-values.yaml`) and edit it in-place. A few important knobs:

```yaml
global:
  publicHost: mip.example.org

platform-ui:
  backend:
    host: platform-backend-service
    port: 8080
    context: services
  notebook:
    enabled: true
  ingress:
    tlsSecretName: platform-ui-tls

keycloak:
  enabled: true
  host: iam.example.org
```

See `values.yaml` for the full `jupyterhub` block (images, hub and notebook resources, storage).

JupyterHub encrypts auth state with the key in the `jupyterhub-crypt` secret. Either set `jupyterhub.cryptKey` to a stable value, or leave it empty and create the secret once, like `keycloak-credentials`:

```
kubectl create secret generic jupyterhub-crypt -n <namespace> --from-literal=crypt-key=$(openssl rand -hex 32)
```

The chart never generates the key itself, so it stays stable across ArgoCD syncs and `helm template` renders.

Register a Keycloak redirect URI for the Hub OAuth client:

`https://<global.publicHost>/notebook/hub/oauth_callback`

For Kubernetes, make sure the images set in `jupyterhub.image` and `jupyterhub.singleuser.image` are pushed to the configured registry or loaded onto every node that may run the pods.

`jupyterhub.spawner: operator-api` removes the hub's Kubernetes access entirely. The hub creates and deletes notebooks through the notebook operator's HTTPS spawn API (`jupyterhub.operatorApi.url`, which must start with `https://`) instead of the Kubernetes API, and the chart renders no Role or RoleBinding, whatever `jupyterhub.rbac.create` says — with `jupyterhub.rbac.create: false`, remove the hand-applied Role and RoleBinding after switching. Both the `jupyterhub` ServiceAccount and the hub pod set `automountServiceAccountToken: false`, so the hub pod gets no ServiceAccount token at the standard path; it authenticates with a projected token bound to the audience `mip-notebook-operator`, which the kubelet rotates and the API server rejects, and trusts the spawn API's CA from the `mip-notebooks-ca` ConfigMap (`jupyterhub.operatorApi.caConfigMap`), which mip-infra copies into each federation. The hub picks up a rotated token or CA without a restart. This mode needs `hbpmip/mip-jupyterhub` `0.3.0` or later (older images do not know `operator-api` and fall back to KubeSpawner without credentials) and the spawn API deployed by mip-infra.

The reachability diagram from the legacy profiles is still valid as a reference for deciding the correct public URL:
![MIP Reachability Scheme](../docs/MIP_Configuration.png)

`global.publicHost` defaults to `hbpmip.link`. Override it in your custom values file or with `--set-string global.publicHost=<hostname>` whenever a deployment needs a different public hostname.

`global.publicHost` must be the bare hostname served by the ingress, without `http://` or `https://`.

`keycloak.protocol` defaults to `https` and `keycloak.realm` defaults to `MIP`. Override them only when your external Keycloak differs from those defaults.

If you deploy behind a reverse proxy or load balancer, forward the standard `X-Forwarded-*` headers to the backend so Spring can reconstruct the public request URL correctly.

**WARNING!**: In **ANY** case, when you use an **EXTERNAL** KeyCloak service (i.e. iam.ebrains.eu), make sure that you use the correct *CLIENT_ID* and *CLIENT_SECRET* to match the MIP instance you're deploying!

After tailoring `values.yaml` you can deploy the UI Helm chart. We still recommend deploying the engine Helm charts first, before installing the UI components.

### Using MIP from users' own computers

The `mip` Python client signs in through a **public** Keycloak client of its own, one per federation, e.g. `mip-cli-<federation>`. Users create a token on the portal's account page, or run `mip login`, and the client renews access with Keycloak by itself. Set the client's ID in `keycloak.cliClientId`; leave it empty to keep the feature off (the account page then hides the guide).

The client needs, in Keycloak: client authentication off; standard flow on with PKCE `S256` **enforced** (not an empty challenge method), redirect URI exactly `https://<global.publicHost>/account` (no wildcard) and web origin `https://<global.publicHost>`; the OAuth 2.0 Device Authorization Grant on; implicit flow, direct access grants and service accounts **off**; *Full scope allowed* **off**, with the platform client's data roles (or one composite role holding them all) added to the local client's dedicated scope instead — whenever a new dataset role is created on the platform client, add it there too, or local-client users silently lose that dataset; the `roles` client scope set to *Optional* (so other clients' and realm roles stay out of its tokens); a copy of the federation client's *User Client Role* mapper (client `KEYCLOAK_CLIENT_ID`, claim `authorities`); an *Audience* mapper for that client; the `offline_access` scope; a *Client Offline Session Idle* of 30 days and a *Client Offline Session Max* of 180 days. Keycloak applies offline maximums only when the realm's *Offline Session Max Limited* is on, and caps the client's at the realm's *Offline Session Max*; to leave the realm's other clients unaffected, set that realm maximum high (e.g. 3650 days). Without it, only the offline idle timeout ever signs a computer out. Also ask EBRAINS for an authentication flow override on the local client without the Cookie execution (every sign-in then needs the password, whatever the request asks) and for *Consent required* with a consent text naming the client. Users and roles stay on the federation's client.

### Ending a local-client sign-in

Revoke a computer's access through Keycloak; the next `mip` call then fails when it tries to renew.

*As the user:* open the Keycloak account console at `https://<keycloak host>/realms/<realm>/account`, go to *Applications*, find `mip-cli-<federation>` and choose *Remove access*; this ends the offline session. On the local machine, `mip logout` clears the stored token.

*As an administrator:* in *Users*, open the user, then *Sessions* to sign out the offline sessions and *Consents* to revoke the client's consent. Disabling the user also stops renewal.

*Lost device:* revoke as above, have the user change their password if the machine may have been unlocked, and tell the federation operator. After revocation, an access token already issued still works for up to 5 minutes.

For the administrator steps at EBRAINS, contact the realm administrators; record the contact and expected response time here once confirmed. When a new dataset role is created on the platform client, add it to the local client's scope as well (see the settings above), or local-client users silently lose that dataset.

Before turning the feature on (setting `keycloak.cliClientId`), check off:

- [ ] The Keycloak client settings above are verified: the sign-in settings, the federation-limited scope, and the re-authentication override and consent (ask EBRAINS).
- [ ] `keycloak.requiredAudience` is set on the backend (step 5 under *Security settings for each federation*).
- [ ] A platform-ui image with `prompt=login` on the account page is deployed (name the minimum version here once released).
- [ ] The revocation runbook above (*Ending a local-client sign-in*) is in place.


### Security settings for each federation

Do these in order; each step needs the one before it.

1. **Deploy a backend with the token checks everywhere first.** Older backends accept any token of the realm, so one federation's token reads another's data. Local-client sign-in stays off until this is done.
2. **Give each federation its own platform client.** Federations sharing a client share tokens: their backends accept each other's. This needs a new client in Keycloak, with users and roles assigned to it.
3. **Set the platform client's *Backchannel Logout URL*** to `https://<global.publicHost>/services/logout/connect/back-channel/keycloak`. Without it, a disabled user keeps a portal session for up to 8 hours.
4. **Keep notebook pods away from Exaflow and the backend database.** Exaflow has no user authentication, and notebook pods run users' code. This chart ships no NetworkPolicy: `mip-infra` limits the notebooks' egress to the hub API, the backend API and DNS (`common/security/federation/templates/notebook-network-policy.yaml`). Clusters without it need an equivalent policy, on a CNI that enforces NetworkPolicies. A policy on Exaflow's pods does not help if another policy opens them to the namespace. From a notebook terminal, check that each of these times out; a name resolution error proves nothing:
   ```bash
   curl -m 5 http://exaflow-controller-service:5000/healthcheck
   python -c "import socket; socket.create_connection(('exaflow-workers-service', 5672), 5)"
   python -c "import socket; socket.create_connection(('exaflow-aggregation-service', 50051), 5)"
   ```
5. **Set `keycloak.requiredAudience`** to the platform client once its portal, notebook and local-client tokens all carry it in `aud`. Keycloak does not add it by default: add an *Audience* mapper for the platform client to each client that issues these tokens. Decode one of each first — for a local-client token, check `azp` (the local client), `aud` (contains the platform client) and `typ` (`Bearer`) — a wrong value refuses every API call. Local-client sign-in (`keycloak.cliClientId`) stays off until this is set.

Upgrade notes:

- Users without a role on the platform client can no longer open notebooks, and users without a shared experiment's datasets can no longer read it.
- The chart does not change image tags. `KEYCLOAK_REQUIRED_AUDIENCE` and `KEYCLOAK_ORIGIN` do nothing until you deploy images that read them.

### Upgrading from 9.2.x

1. Back up the platform database. platform-backend 11.0.0 runs Flyway V5 and V6 on startup, and V6 renames tables and columns.
2. Create the `exaflow-rabbitmq` Secret (see *Analyses through RabbitMQ*).
3. Update the Exaflow values: rename `localnodes` to `worker_count`, and remove global-worker, Flower and SMPC settings.
4. Upgrade Exaflow 1.3.0, platform-backend 11.0.0, platform-ui 3.0.0 and mip-jupyter 0.3.0 together.
5. Rename `EXPERIMENT_RUN_DRAFT_TTL` to `ANALYSIS_RUN_DRAFT_TTL` if you set it.
6. Give every notebook user a role on the platform's Keycloak client. Users without one get a 403.
7. External API clients must move from `/experiments` to `/analyses`.

### Microk8s installation
On a running Ubuntu (we recommend 22.04) distribution, install microk8s (we **HIGHLY** recommend to **NOT** install Docker on your Kubernetes cluster!):
```
sudo snap install microk8s
```
```
sudo adduser mipadmin
```
```
sudo adduser mipadmin sudo
```
```
sudo adduser mipadmin microk8s
```

As *mipadmin* user:
```
microk8s enable dns helm3 ingress storage
```
```
sudo mkdir -p /data/<MIP_INSTANCE_OR_FEDERATION_NAME>
```
```
sudo chown -R mipadmin.mipadmin /data
```

The web-app chart uses dynamically provisioned PVCs on microk8s too. The `storage` addon provides the `k8s.io/microk8s-hostpath` provisioner, and the chart bootstraps a local `StorageClass` on top of it for the MIP PVCs. That `StorageClass` uses `WaitForFirstConsumer` binding so PVC placement follows pod scheduling more safely on multi-node microk8s clusters.

On microk8s, the MIP web-app chart schedules both `platform-backend` and `platform-ui` onto nodes labeled `master=true`, so make sure the node intended to host the stack carries that label.

For a "federated" deployment, you may want to add nodes to your cluster. "microk8s add-node" will give you a **one-time usage** token, which you can use on a worker node to actually "join" the cluster. This process must be repeated on all the worker nodes.

### Exaflow deployment

Clone the matching engine release and create an environment-specific values file:

```bash
sudo git clone --branch 1.3.0 https://github.com/madgik/exaflow /opt/exaflow
sudo chown -R mipadmin:mipadmin /opt/exaflow
cp /opt/exaflow/kubernetes/values.yaml /opt/exaflow/kubernetes/my-values.yaml
```

Set these values in `my-values.yaml` for a microk8s deployment. Replace the
namespace, federation name, paths, and worker count for your installation:

```yaml
namespace: <target-namespace>
managed_cluster: false
worker_count: 1
exaflow_images:
  repository: madgik
  version: 1.3.0
federation: <MIP_INSTANCE_OR_FEDERATION_NAME>
aggregation_server:
  enabled: true
rabbitmq:
  existing_secret: exaflow-rabbitmq
storage:
  hostPath:
    db:
      localworker: /data/<MIP_INSTANCE_OR_FEDERATION_NAME>/localworker
```

`worker_count` counts workers. The chart requires one worker per
eligible host through pod anti-affinity; a single-host installation uses `1`
and labels the same node as both master and worker. The current templates use
`managed_cluster` to select hostPath versus CSV PVC storage; `storage.type`
alone does not change that selection. For managed clusters, set
`managed_cluster: true` and configure `storage.cephfs.storageClassName` and the
CSV volume sizes in the Exaflow values file.

Label the nodes for an unmanaged installation:

```bash
microk8s kubectl label node <MASTER_HOSTNAME> master=true
microk8s kubectl label node <WORKER_HOSTNAME> worker=true
```

After preparing the dataset directories or volumes, deploy:

```bash
microk8s helm3 upgrade --install exaflow /opt/exaflow/kubernetes \
  --namespace <target-namespace> --create-namespace \
  -f /opt/exaflow/kubernetes/my-values.yaml
```

Keep the Helm namespace and the Exaflow `namespace` value identical. The
backend's `engines.exaflow.url` must reach `exaflow-controller-service`; the
existing short service name works when both charts share a namespace. For
separate namespaces use the controller's fully qualified service DNS name.

See the [Exaflow chart](https://github.com/madgik/exaflow/tree/1.3.0/kubernetes)
for the full values and templates. The old `credentials_location`,
`db.storage_location`, `db.csvs_location`, and `controller.cleanup_file_folder`
settings are not used by this chart.

### Analyses through RabbitMQ

MIP submits analyses to Exaflow over RabbitMQ (specifications and metadata still use `engines.exaflow.url`). The chart always deploys the in-namespace RabbitMQ 4.4.1 broker.

1. Use Exaflow 1.3.0 or later and platform-backend 11.0.0 or later. Upgrade the controller, workers, and aggregation server together. Run one Exaflow controller replica and one backend replica.
2. Create the `exaflow-rabbitmq` Secret before deploying. Use a URL-safe password (`openssl rand -hex 24`): the same password is embedded in `url`, which Exaflow and the backend read, while the broker reads `username` and `password`. Keep `%2F` as the trailing vhost path so both the Java and Python AMQP clients select the default `/` vhost. The host is the in-namespace Service:

```bash
RABBITMQ_PASSWORD=$(openssl rand -hex 24)
kubectl create secret generic exaflow-rabbitmq -n <namespace> \
  --from-literal=username=mip \
  --from-literal=password="$RABBITMQ_PASSWORD" \
  --from-literal=url="amqp://mip:${RABBITMQ_PASSWORD}@exaflow-rabbitmq:5672/%2F"
```

3. Deploy this chart so the RabbitMQ broker starts.
4. The Exaflow chart requires the broker Secret. Its defaults `existing_secret: exaflow-rabbitmq` and `url_secret_key: url` already match this Secret. Deploy Exaflow. `GET /healthcheck/rabbitmq` on the controller must return 200 with `consuming: true`.
5. Delivery is at least once, so an analysis can run twice after a controller crash. Broker persistence and backups are your responsibility. The StatefulSet retains its PVC when the broker is deleted or scaled down; keep that PVC until dead-letter messages are inspected.
6. `RABBITMQ_DEFAULT_USER` and `RABBITMQ_DEFAULT_PASS` apply only on first boot with an empty volume. Rotate by running `kubectl exec -n <namespace> exaflow-rabbitmq-0 -- rabbitmqctl change_password <user> <new-password>` first, then update the Secret (`password` and `url`), then restart the Exaflow controller and platform-backend.
7. This chart's broker is RabbitMQ 4.4.1 and writes `consumer_timeout = 7200000`, `max_message_size = 33554432`, `vm_memory_high_watermark.relative = 0.6`, and `raft.wal_max_size_bytes = 134217728` into `rabbitmq.conf` with a 2Gi memory limit. Keep those settings if tuning overrides: RabbitMQ's 30 minute consumer timeout and 16 MiB message limit make long analyses redeliver and oversized requests fail at the broker instead of returning Exaflow's `request_too_large`, and less memory risks tripping the memory alarm or OOMKills under the quorum Raft WAL, 32 MiB messages, and 256 MiB queues.
8. Optional hardening: the Secret's user is the broker administrator and is shared by Exaflow and the backend. For stricter isolation, create a backend user, put its URL in its own Secret, and point platform-backend at that Secret with `engines.exaflow.rabbitmq.existingSecret` and `engines.exaflow.rabbitmq.urlSecretKey`. Leave `existingSecret` empty to keep using `rabbitmq.existingSecret` and the `url` key. On RabbitMQ 4.3.1 and later, including 4.4.1, `queue.declare(passive=true)` needs at least one permission (`read`, `write`, or `configure`) on `exaflow.analysis.requests`. Grant the backend user `configure` `^platform-backend\.analysis\.replies$`, `write` `^(amq\.default|platform-backend\.analysis\.replies)$`, and `read` `^(platform-backend\.analysis\.replies|exaflow\.analysis\.requests)$`.

### Web and notebook stack deployment

The deployment chart lives in this **mip** repository:

```bash
sudo git clone https://github.com/Medical-Informatics-Platform/mip /opt/mip
sudo chown -R mipadmin:mipadmin /opt/mip
cp /opt/mip/deployment/kubernetes/values.yaml /opt/mip/deployment/kubernetes/my-values.yaml
```

Use the MIP 9.3.0 release checkout when available. Edit `my-values.yaml` for your
installation, including image versions, ingress settings, storage classes, and
external authentication. On microk8s set `cluster.managed: false`; on managed
clusters set it to `true` and choose the appropriate managed storage class.
The MIP `cluster.managed` flag is independent of Exaflow's `managed_cluster`.

Provision the referenced `mip-secret`, `exaflow-rabbitmq`, `keycloak-credentials`, TLS secret, and
`jupyterhub-crypt` through your secret-management process. Required secret keys
are declared in the chart templates; keep private values out of the repository.
For notebooks, preserve the crypt key across deployments and configure the
Keycloak callback described above.

```bash
microk8s helm3 upgrade --install mip /opt/mip/deployment/kubernetes \
  --namespace <target-namespace> --create-namespace \
  -f /opt/mip/deployment/kubernetes/my-values.yaml
```

Confirm that the workloads are ready, the portal can discover data and run an
analysis, and a notebook session can connect to the backend. Use
[Backup and Recovery](../docs/BackupAndRecovery.md) before upgrades or storage
changes.

## Recovery expectations

Kubernetes may restart or reschedule workloads after a failure, but that does
not recover lost database contents, notebook files, secrets, or node-local
storage. Recovery time depends on the failure and storage backend. Maintain
verified backups and follow the [recovery procedure](../docs/BackupAndRecovery.md)
rather than relying on automatic workload restart as a backup strategy.
