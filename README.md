# Technical Documentation for the Medical Informatics Platform (MIP)  <!-- omit in toc --> 


A powerful federated data processing and analysis system that preserves patient privacy. More info on the [MIP Website](https://ebrains.eu/data-tools-services/medical-analytics/medical-informatics-platform) 

# Table of Content <!-- omit in toc --> 

- [Preamble](#preamble)
- [9.1 Release](#mip-91-release--major-updates)
- [Components](#components)
- [Deployment](#deployment)
- [Federated Analysis Algorithms](#federated-analysis-algorithms)
- [Data Management](#data-management)
- [High Level Description](#high-level-description)
- [Architecture](#architecture)

# Preamble

This repository is an index for a collection of documents and other sources of information related to the Medical Informatics Platform. The intended audience comprises developers, technical deployment and support teams, and anyone else with a deep technical interest in the functioning of the MIP. Its purpose is to facilitate access to a range of information necessary to represent the current state of the MIP. It should provide adequate material for suitably qualified staff to understand how the MIP works, and to develop, deploy and operate the MIP.

This information is evolving along with the MIP so please make sure you consult the document version that is relevant to the indented, or preferably, latest version of the Medical Informatics Platform.

In the following sections, links and references to useful information is made available.

# MIP 9.1 Release – Major Updates

## **Broader federated analytics**

MIP 9.1 expands the range of analyses that can be performed across distributed
clinical datasets without moving patient-level data. The release adds support
for more advanced statistical workflows, including survival analysis,
association testing, histogram exploration, outlier reporting, and mixed-effects
modeling, while refreshing the documentation for the core analysis portfolio.

## **Improved analysis workflow**

The analysis experience has been refined to make experiment setup and result
review clearer. Users get better guidance when selecting variables and
configuring algorithms, clearer feedback when an analysis cannot run with the
selected inputs, and improved result views with more consistent tables, charts,
labels, and export actions.

## **Built-in data preparation options**

MIP 9.1 introduces more data preparation options directly into the analysis
workflow. Missing-value handling, outlier handling, and longitudinal
transformations can be configured before running an analysis, reducing the need
for manual preparation outside the platform.

## **More maintainable and deployment-ready platform**

The platform has been updated with newer component versions, refreshed
deployment defaults, and reviewed documentation for deployment, data management,
architecture, and onboarding. The release also improves alignment between the
user interface, backend services, and federated analysis engine, making the
platform easier to operate and evolve.

# Components

The main [MIP building blocks](documentation/Components.md) are listed along with the respective repositories that host them.

# Deployment

The MIP comes with a single code base but with two modes of deployment. One for local usage only, and one that enables the creation of a federation of nodes. Information on the different deployment approaches can be found in the following location:

- [Deployment Documentation](deployment)

# Federated Analysis Algorithms

This includes documentation on existing algorithm federation approach as well as information related to creating a new algorithm.

- [Available federated analysis algorithms](documentation/algorithms.md)
- [Exaflow Analytic Engine](https://github.com/madgik/exaflow/tree/1.1.0)

# Data Management

For all details relating to the Data Factory, how to manage your data and process it for use with the MIP, please consult the following document

- [Data Management Guide](documentation/MIP_Data_management_documentation.md)

A detailed user guide for Data Quality Control tool can be found here:
 - [Data Quality Control Tool Guide](https://github.com/HBPMedical/DataQualityControlTool/wiki)

Data Catalog is a component of the Medical Informatics Platform (MIP) for the EBrains. It enables seamless management, visualization, and access to data models and medical conditions.
 - [Data Catalog Guide](https://github.com/Medical-Informatics-Platform/datacatalog)

# High Level Description

For a high-level description of the MIP please consult:

- [The MIP: A powerful federated data processing and analysis system that preserves patient privacy](https://ebrains.eu/data-tools-services/medical-analytics/medical-informatics-platform) on EBRAINS research infrastructure

# Architecture

- [High-level view of the architecture](documentation/Architecture.md), the main building blocks and data flows.

# Onboarding guidance

- [Onboarding to the Medical Informatics Platform MIP](https://wiki.ebrains.eu/bin/view/Collabs/onboarding-to-the-mip/) on EBRAINS Collaboratory

# Acknowledgement
This project/research received funding from the European Union’s Horizon 2020 Framework Programme for Research and Innovation under the Framework Partnership Agreement No. 650003 (HBP FPA).
