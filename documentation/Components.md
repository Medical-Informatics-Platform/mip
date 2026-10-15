# Components in MIP 9.2.1

MIP combines the following independently released repositories. These component
versions make up the MIP 9.2.1 release.

| Repository | Version | Role and major changes |
| --- | --- | --- |
| [platform-ui](https://github.com/Medical-Informatics-Platform/platform-ui/tree/2.1.0) | 2.1.0 | Redesigned Experiment Studio and Data Handling, filter-based categorical columns, K-means preprocessing, refreshed visualizations, experiment folders, exports, and notebook entry point. Saving an experiment stores the reviewed run without executing it again. |
| [platform-backend](https://github.com/Medical-Informatics-Platform/platform-backend/tree/10.1.0) | 10.1.0 | Analysis and specification endpoints, experiment folder persistence and management, authenticated access for web and notebook analyses, and kept runs that can be saved without re-execution. |
| [exaflow](https://github.com/madgik/exaflow/tree/1.2.1) | 1.2.1 | Federated execution and preprocessing, reusable K-means cluster columns, quartiles, binned Mann-Whitney U, Standardized Mean Difference, and updated analysis requests. |
| [mip-jupyter](https://github.com/Medical-Informatics-Platform/mip-jupyter/tree/0.2.0) | 0.2.0 | JupyterLab workspace, `mip` Python client (distributed as `mip-client`), welcome notebook, JupyterHub and single-user images, and Cohort Scout notebook support agent. Example notebooks now come from the shared, read-only [mip-examples](https://github.com/Medical-Informatics-Platform/mip-examples) repository. |

## Deployment and infrastructure

- [Deployment package](../deployment): local and federated installation tooling
  in this repository.
- [MIP Infrastructure](https://github.com/Medical-Informatics-Platform/mip-infra):
  MIP KaaS infrastructure based on ArgoCD deployments.
- [Notebook integration notes](https://github.com/Medical-Informatics-Platform/mip-jupyter/blob/0.2.0/docs/operators.md):
  JupyterHub, authentication, workspace images, and assistant configuration.

The backend 10.x and Exaflow 1.2.x releases change the analysis request/API
contracts. Integrations must use the revised analysis and specification APIs;
older clients and deployment configurations need review before upgrading.
Notebook access requires JupyterHub deployment and authentication wiring, and
Cohort Scout requires its assistant service to be configured.

## Additional standalone tools

- [Quality Control Tools](https://github.com/HBPMedical/DataQualityControlTool/tree/v.4.1)
- [DataCatalog](https://github.com/Medical-Informatics-Platform/datacatalog/tree/2.0.0)
