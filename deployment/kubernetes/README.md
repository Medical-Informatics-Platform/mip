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

### The engine: [Exaflow](https://github.com/madgik/exaflow/tree/master/kubernetes)
* [controller](https://github.com/madgik/exaflow/tree/master/exaflow/controller)
* [worker](https://github.com/madgik/exaflow/tree/master/exaflow/worker)

* [smpc-db](https://github.com/docker-library/mongo)
* [smpc-queue](https://github.com/docker-library/redis)
* [smpc-coordinator](https://github.com/Exaflow/tree/master/exaflow)
* [smpc_player](https://github.com/Exaflow/tree/master/exaflow)
* [smpc-client](https://github.com/madgik/exaflow/tree/master/exaflow)

### The web app stack:
* [platform-ui](https://github.com/Medical-Informatics-Platform/platform-ui): The "Web App" UI
* [platform-backend](https://github.com/Medical-Informatics-Platform/platform-backend): The "Backend API" which supports the Web App
    * Its database bootstrap script lives next to the application code (`config/scripts/bootstrap-platform-backend-db.sh`) and the same script is vendored in this chart under `files/platform-backend-db-init.sh` so the deployment can mount it via ConfigMap without embedding a large shell block inside the template. Keeping both copies in sync lets the container image and the Helm release evolve together.
* [platform_backend_db](https://github.com/docker-library/postgres): The platform-backend's database
**External Keycloak**: Authentication is provided by an existing Keycloak realm; this chart only wires the configuration values so the UI stack can reach it.


## Taking care of the medical data
### Storing the data in the worker VMs
On each **worker** node, a folder should be created `/data/<MIP_INSTANCE_OR_FEDERATION_NAME>/<PATHOLOGY_NAME>` for
every pathology for which we will have at least one dataset.
Afterward, The dataset CSV files should be placed in their proper pathology folder.

## Configuration
Prior to deploying it (on a microk8s K8s cluster of one or more nodes), there are a few adjustments to make in `values.yaml`. Each top-level section controls a part of the stack:

* `cluster`: storage classes and whether the chart should use the managed storage class or the microk8s local storage class.
* `global`: shared public hostname used by the ingress and backend redirects.
* `platform-ui`, `platform-backend`, `platformBackendDatabase`: container images and component specific options (including the shared ingress/tls settings and PVC sizes).
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
  ingress:
    tlsSecretName: platform-ui-tls

keycloak:
  enabled: true
  host: iam.example.org
```

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

For a "federated" deployment, you may want to add nodes to your cluster. "microk8s add-node" will give you a **one-time usage** token, which you can use on a worker node to actually "join" the cluster. This process must be repeated on all the worker nodes.

### Exaflow Deployment
* Install the repository content
  ```
  sudo git clone https://github.com/madgik/exaflow /opt/exaflow
  ```
  ```
  sudo chown -R mipadmin.mipadmin /opt/exaflow
  ```
* Set the variables in /opt/exaflow/kubernetes/values.yaml
    * localnodes: 1 for a "local" deployment (yes, even if it's the same machine for master and worker), or more (the number of workers, not counting the master node) for a "federated" deployment
    * credentials_location: /opt/exaflow/credentials
    * db.storage_location: /opt/exaflow/.stored_data/db
    * db.csvs_location: /data/<MIP_INSTANCE_OR_FEDERATION_NAME>
    * controller.cleanup_file_folder: /opt/exaflow/.stored_data/cleanup
    * smpc.enabled: true (if you want, and **ONLY** in case of a federated deployment, and also **ONLY** if you have at least 3 worker nodes!)
* Label the nodes

  For the master node:
  ```
  microk8s kubectl label node <MASTER_HOSTNAME> master=true
  ```
  For all the worker nodes (even on a "local" deployment where the master and the worker are the **same** machine), add *worker* and (if you want) *smpc_player* labels:
  ```
  microk8s kubectl label node <WORKER_HOSTNAME> worker=true
  ```
  ```
  microk8s kubectl label node <WORKER_HOSTNAME> smpc_player=true
  ```
* Deploy the Helm chart
  ```
  microk8s helm3 install exaflow /opt/exaflow/kubernetes
  ```

For a more in-depth guide on deploying Exaflow, please refer to the documentation available on the [Exaflow Kubernetes repository](https://github.com/madgik/exaflow/blob/master/kubernetes).



### Web App Stack Components Deployment
* Install the repository content
  ```
  sudo git clone https://github.com/Medical-Informatics-Platform/mip-deployment /opt/mip-deployment
  ```
  ```
  sudo chown -R mipadmin.mipadmin /opt/mip-deployment
  ```

* Copy `values.yaml` to `/opt/mip-deployment/kubernetes/my-values.yaml` and tailor it to your environment.
  * On microk8s, keep `cluster.managed: false` so the chart uses the local storage class it bootstraps.
* Deploy (or upgrade) the Helm release with your customised values
  ```
  microk8s helm3 upgrade --install mip \
    --namespace <target-namespace> \
    -f /opt/mip-deployment/kubernetes/my-values.yaml \
    /opt/mip-deployment/kubernetes
  ```

# MicroK8s Automatic Recoverability

## Overview
MicroK8s is designed for simplicity and resilience. One of its key features is the automatic recoverability of both federated clusters and individual local nodes.

## Automatic Recoverability in Federation
In a federated cluster setup, MicroK8s ensures high availability and fault tolerance. If the master node in a federation faces downtime or operational issues, MicroK8s is designed to automatically recover its state and functionality.

### Key Points:
- **Self-healing Mechanism**: MicroK8s employs a self-healing mechanism that triggers upon detecting issues with the master node.
- **State Restoration**: Automatically restores the master node to its last known healthy state without manual intervention.

## Local Node Recoverability
For individual local nodes, MicroK8s offers a robust recovery process. This process is vital in scenarios where local nodes experience disruptions.

### Key Points:
- **Node Health Monitoring**: Continuous monitoring of node health to quickly identify any disruptions.
- **Automatic Restoration**: On reboot or reconnection, the local node automatically synchronizes and restores its state to align with the federation's current status.

## Recovery Time Frame
The recovery process in MicroK8s, whether for a federation or a local node, typically completes within a brief period.

### Expected Timeline:
- **Minimum Recovery Time**: Approximately 1 minute.
- **Maximum Recovery Time**: Up to 5 minutes, depending on the complexity and scale of the cluster.
