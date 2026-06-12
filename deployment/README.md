# Deployment of the Medical Informatics Platform (MIP)

This folder contains the deployment and operations documentation for MIP.

## Main Deployment Guides

### Development Deployment
Use the development guide for local/non-production setup with docker-compose.

- [Development deployment guide](dev/README.md)
- Includes prerequisites (Python, docker-compose), startup instructions, basic tests, and shutdown steps.

### Kubernetes Deployment
Use the Kubernetes guide for production-like and federated installations.

- [Kubernetes deployment guide](kubernetes/README.md)
- Includes infrastructure modes (microk8s VMs vs managed clusters), hardware/software requirements, component overview, data placement, deployment steps, and recovery behavior.

## Supporting Documentation

- [Data requirements for onboarding new datasets](../documentation/MIP_Data_management_documentation.md): CSV and `CDEsMetadata.json` format rules, including additional constraints for longitudinal data.
