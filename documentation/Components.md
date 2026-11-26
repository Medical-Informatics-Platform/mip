# Components in MIP 9.3.0

MIP combines the following independently released repositories. These component
versions make up the MIP 9.3.0 release.

| Repository | Version | Role and major changes |
| --- | --- | --- |
| [platform-ui](https://github.com/Medical-Informatics-Platform/platform-ui/tree/3.0.0) | 3.0.0 | Experiments are now analyses and studies. New data exploration and result views, access tokens for the `mip` client on users' own computers, and an enforced Content-Security-Policy. |
| [platform-backend](https://github.com/Medical-Informatics-Platform/platform-backend/tree/11.0.0) | 11.0.0 | `/analyses` and `/studies` APIs, analyses submitted over RabbitMQ, and hardened sign-in, sessions and data access. |
| [exaflow](https://github.com/madgik/exaflow/tree/1.3.0) | 1.3.0 | RabbitMQ analysis submission, FastAPI controller, PCA components as downstream inputs, new federated preprocessing steps, and unified workers (no global worker, Flower or SMPC). |
| [mip-jupyter](https://github.com/Medical-Informatics-Platform/mip-jupyter/tree/0.3.0) | 0.3.0 | JupyterLab workspace and `mip` Python client (distributed as `mip-client`) with `mip login`, the `mip` command, JupyterHub and single-user images, and the Cohort Scout support agent. |

## Deployment and infrastructure

- [Deployment package](../deployment): local and federated installation tooling
  in this repository.
- [MIP Infrastructure](https://github.com/Medical-Informatics-Platform/mip-infra):
  MIP KaaS infrastructure based on ArgoCD deployments.
- [Notebook integration notes](https://github.com/Medical-Informatics-Platform/mip-jupyter/blob/0.3.0/docs/operators.md):
  JupyterHub, authentication, workspace images, and assistant configuration.

platform-ui 3.0.0, mip-jupyter 0.3.0 and platform-backend 11.0.0 use new API
routes. Deploy them together with Exaflow 1.3.0. Older clients do not work with
this backend. See [Upgrading from 9.2.x](../deployment/kubernetes/README.md#upgrading-from-92x).
Notebook access requires JupyterHub deployment and authentication wiring, and
Cohort Scout requires its assistant service to be configured.

## Additional standalone tools

- [Quality Control Tools](https://github.com/HBPMedical/DataQualityControlTool/tree/v.4.1)
- [DataCatalog](https://github.com/Medical-Informatics-Platform/datacatalog/tree/3.0.0)
