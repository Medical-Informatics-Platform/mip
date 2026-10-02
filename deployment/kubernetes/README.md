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

Deploy the [Exaflow 1.2.1 chart](https://github.com/madgik/exaflow/tree/1.2.1/kubernetes)
for the native federated pipeline:

- Controller: accepts analyses and coordinates execution.
- Local workers: load each site's datasets and execute local computations.
- Global worker: provides the chart's global-worker service.
- Aggregation server: combines partial vectors for algorithms and preprocessing
  steps that require aggregation. Keep `aggregation_server.enabled: true` for
  the full native algorithm portfolio.

Flower and SMPC are optional features with separate setup requirements. The
example below disables them and focuses on the native pipeline.

### The web and notebook stack

- **platform-ui**: Experiment Studio, result visualizations, and notebook entry.
- **platform-backend**: authenticated analysis API and experiment management.
- **PostgreSQL**: stores platform experiments, folders, and other application state.
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
* `keycloak`: toggles the connection parameters to the external Keycloak instance (`enabled`, `host`, `protocol`, `realm`).

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

The reachability diagram from the legacy profiles is still valid as a reference for deciding the correct public URL:
![MIP Reachability Scheme](../docs/MIP_Configuration.png)

`global.publicHost` defaults to `hbpmip.link`. Override it in your custom values file or with `--set-string global.publicHost=<hostname>` whenever a deployment needs a different public hostname.

`global.publicHost` must be the bare hostname served by the ingress, without `http://` or `https://`.

`keycloak.protocol` defaults to `https` and `keycloak.realm` defaults to `MIP`. Override them only when your external Keycloak differs from those defaults.

If you deploy behind a reverse proxy or load balancer, forward the standard `X-Forwarded-*` headers to the backend so Spring can reconstruct the public request URL correctly.

**WARNING!**: In **ANY** case, when you use an **EXTERNAL** KeyCloak service (i.e. iam.ebrains.eu), make sure that you use the correct *CLIENT_ID* and *CLIENT_SECRET* to match the MIP instance you're deploying!

After tailoring `values.yaml` you can deploy the UI Helm chart. We still recommend deploying the engine Helm charts first, before installing the UI components.


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
sudo git clone --branch 1.2.1 https://github.com/madgik/exaflow /opt/exaflow
sudo chown -R mipadmin:mipadmin /opt/exaflow
cp /opt/exaflow/kubernetes/values.yaml /opt/exaflow/kubernetes/my-values.yaml
```

Set these values in `my-values.yaml` for a microk8s deployment. Replace the
namespace, federation name, paths, and worker count for your installation:

```yaml
namespace: <target-namespace>
managed_cluster: false
localnodes: 1
exaflow_images:
  repository: madgik
  version: 1.2.1
federation: <MIP_INSTANCE_OR_FEDERATION_NAME>
aggregation_server:
  enabled: true
flower:
  enabled: false
smpc:
  enabled: false
storage:
  hostPath:
    db:
      localworker: /data/<MIP_INSTANCE_OR_FEDERATION_NAME>/localworker
      globalworker: /data/<MIP_INSTANCE_OR_FEDERATION_NAME>/globalworker
```

`localnodes` counts local workers. The chart requires one local worker per
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

See the [Exaflow chart](https://github.com/madgik/exaflow/tree/1.2.1/kubernetes)
for the full values and templates. The old `credentials_location`,
`db.storage_location`, `db.csvs_location`, and `controller.cleanup_file_folder`
settings are not used by this chart.

### Web and notebook stack deployment

The deployment chart lives in this **mip** repository:

```bash
sudo git clone https://github.com/Medical-Informatics-Platform/mip /opt/mip
sudo chown -R mipadmin:mipadmin /opt/mip
cp /opt/mip/deployment/kubernetes/values.yaml /opt/mip/deployment/kubernetes/my-values.yaml
```

Use the MIP 9.2.0 release checkout when available. Edit `my-values.yaml` for your
installation, including image versions, ingress settings, storage classes, and
external authentication. On microk8s set `cluster.managed: false`; on managed
clusters set it to `true` and choose the appropriate managed storage class.
The MIP `cluster.managed` flag is independent of Exaflow's `managed_cluster`.

Provision the referenced `mip-secret`, `keycloak-credentials`, TLS secret, and
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
