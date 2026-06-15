# Medical Informatics Platform (MIP)  <!-- omit in toc --> 


A federated analytics platform for clinical research networks that need to
collaborate on sensitive health data without centralizing patient-level records.
More information is available on the [MIP Website](https://ebrains.eu/data-tools-services/medical-analytics/medical-informatics-platform).

# Table of Contents <!-- omit in toc --> 

- [About MIP](#about-mip)
- [9.1 Release](#mip-91-release--major-updates)
- [What MIP Includes](#what-mip-includes)
- [Deployment](#deployment)
- [Federated Analysis Algorithms](#federated-analysis-algorithms)
- [Data Management](#data-management)
- [Architecture](#architecture)
- [Onboarding](#onboarding)

# About MIP

The Medical Informatics Platform enables privacy-preserving analysis across
distributed clinical datasets. Hospitals and data providers keep patient-level
data within their local governance boundaries, while researchers can run
federated statistical and machine-learning analyses across participating sites.

This repository collects the technical, deployment, data-management, and
architecture documentation needed to understand, deploy, and operate MIP.

# MIP 9.1 Release – Major Updates

## **Broader federated analytics**

MIP 9.1 expands the range of analyses that can be performed across distributed
clinical datasets without moving patient-level data. The release adds support for
advanced statistical workflows such as survival analysis, association testing,
histogram exploration, outlier reporting, and mixed-effects modeling, while
refreshing the documentation for the core analysis portfolio.

## **Smoother analysis workflow**

The analysis experience has been refined to make experiment setup and result
review clearer. Users get better guidance when selecting variables, clearer
feedback when an analysis cannot run with the selected inputs, and improved
result views with more consistent tables, charts, labels, and export actions.

## **Built-in data preparation**

MIP 9.1 introduces more data preparation options directly into the analysis
workflow. Missing-value handling, outlier handling, and longitudinal
transformations can be configured before running an analysis, reducing the need
for manual preparation outside the platform.

## **Deployment-ready platform**

The platform has been updated with refreshed deployment defaults and reviewed
documentation for deployment, data management, architecture, and onboarding,
making MIP easier to operate and evolve across research infrastructures.

# What MIP Includes

MIP combines a web interface, backend services, federated analysis engine,
deployment tooling, and supporting data-management tools. The main
[MIP building blocks](documentation/Components.md) are listed with the
repositories that host them.

# Deployment

MIP supports both local development deployments and Kubernetes-based deployments
for production-like or federated installations.

- [Deployment Documentation](deployment)

# Federated Analysis Algorithms

The algorithm documentation describes the available federated analyses and links
to the underlying Exaflow analytic engine documentation.

- [Available federated analysis algorithms](documentation/algorithms.md)
- [Exaflow Analytic Engine](https://github.com/madgik/exaflow/tree/1.1.0)

# Data Management

The data-management documentation explains how datasets and metadata are prepared
for use in MIP.

- [Data Management Guide](documentation/MIP_Data_management_documentation.md)

A detailed user guide for the Data Quality Control tool can be found here:
 - [Data Quality Control Tool Guide](https://github.com/HBPMedical/DataQualityControlTool/wiki)

Data Catalog is a component of MIP for EBRAINS. It enables management,
visualization, and access to data models and medical conditions.
 - [Data Catalog Guide](https://github.com/Medical-Informatics-Platform/datacatalog)

# Architecture

- [High-level view of the architecture](documentation/Architecture.md), the main building blocks and data flows.

# Onboarding

- [Onboarding to the Medical Informatics Platform MIP](https://wiki.ebrains.eu/bin/view/Collabs/onboarding-to-the-mip/) on EBRAINS Collaboratory

# Acknowledgement
This project/research received funding from the European Union’s Horizon 2020 Framework Programme for Research and Innovation under the Framework Partnership Agreement No. 650003 (HBP FPA).
