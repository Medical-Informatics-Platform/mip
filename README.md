# Medical Informatics Platform (MIP)  <!-- omit in toc --> 


A federated analytics platform for clinical research networks that need to
collaborate on sensitive health data without centralizing patient-level records.
More information is available on the [MIP Website](https://ebrains.eu/data-tools-services/medical-analytics/medical-informatics-platform).

# Table of Contents <!-- omit in toc --> 

- [About MIP](#about-mip)
- [9.2 Release](#mip-92-release--major-updates)
- [What MIP Includes](#what-mip-includes)
- [Deployment](#deployment)
- [Federated Analysis Algorithms](#federated-analysis-algorithms)
- [Jupyter Notebooks and Support Agent](#jupyter-notebooks-and-support-agent)
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

# MIP 9.2 Release – Major Updates

## Advanced data handling in the UI

The redesigned Data Handling step makes advanced preparation part of the
Experiment Studio workflow. Researchers can configure cohort filters,
missing-value handling, outlier treatment, and transformations, then review
summaries and charts of the prepared data before choosing an analysis.

## More transformation options

Researchers can create new categorical columns from filter-based rules, defining
categories from conditions on existing variables. K-means is also available as
a preprocessing option: a fitted clustering result can be reused to create a
categorical cluster column for downstream analyses. These options extend the
existing missing-value, outlier, and longitudinal transformations.

## Revamped visualizations and simpler execution

The Experiment Studio guides researchers through data exploration, preparation,
algorithm configuration, execution, and results. Algorithm visualizations and
result tables have been redesigned, with PDF reports and CSV exports. The
updated dashboard supports experiment folders, search, and side-by-side result
comparison.

## Jupyter notebooks for federated research

**Jupyter notebooks are a major new feature in MIP.** Researchers can open a
JupyterLab workspace from the portal and build reproducible analyses in Python.
The pre-installed `mip` client provides data discovery, filters, preprocessing,
and algorithm execution through the platform backend. A welcome notebook,
example analyses, and client documentation help researchers get started while
patient-level records remain at the contributing sites.

The new [mip-jupyter repository](https://github.com/Medical-Informatics-Platform/mip-jupyter/tree/0.1.1)
provides the notebook workspace, Python client, and Jupyter images.

## Support agent for notebook creation and explanation

The notebook workspace includes **Cohort Scout**, a support agent that helps
researchers create and edit notebooks, explain analysis code and results, and
navigate the MIP Python client. It is available through Jupyter AI when the
assistant service is configured for the deployment.

## Expanded analytics and supporting services

Exaflow adds histogram-based quartile estimates, a binned Mann-Whitney U test,
Standardized Mean Difference, and richer K-means reporting and preprocessing.
The backend introduces analysis and specification endpoints and experiment
folder management to support the revised workflow. The component releases are
listed in [MIP building blocks](documentation/Components.md).

# What MIP Includes

MIP combines a web interface, backend services, federated analysis engine,
Jupyter notebooks, deployment tooling, and supporting data-management tools.
The main [MIP building blocks](documentation/Components.md) are listed with the
repositories that host them.

# Deployment

MIP supports both local development deployments and Kubernetes-based deployments
for production-like or federated installations.

- [Deployment Documentation](deployment)

# Federated Analysis Algorithms

The algorithm documentation describes the available federated analyses and links
to the underlying Exaflow analytic engine documentation.

- [Available federated analysis algorithms](documentation/algorithms.md)
- [Exaflow Analytic Engine](https://github.com/madgik/exaflow/tree/1.2.1)

# Jupyter Notebooks and Support Agent

Use the notebook workspace to discover available data, compose preprocessing
pipelines, run federated algorithms, and document results alongside Python code.
Cohort Scout can assist with notebook creation and explanation.

- [Notebook getting-started guide](documentation/notebooks.md)
- [MIP Jupyter workspace and Python client](https://github.com/Medical-Informatics-Platform/mip-jupyter/tree/0.1.1)

# Data Management

Data providers prepare and validate source datasets and metadata before upload.
Researchers configure filters and transformations per analysis in the Experiment
Studio or notebook pipelines; these operations do not rewrite uploaded files.

- [Dataset onboarding: Data Management Guide](documentation/MIP_Data_management_documentation.md)

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
