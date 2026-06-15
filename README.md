# Technical Documentation for the Medical Informatics Platform (MIP)  <!-- omit in toc --> 


A powerful federated data processing and analysis system that preserves patient privacy. More info on the [MIP Website](https://ebrains.eu/data-tools-services/medical-analytics/medical-informatics-platform) 

# Table of Content <!-- omit in toc --> 

- [Preamble](#preamble)
- [9.0 Release](#mip-90-release--major-updates)
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

# MIP 9.0 Release – Major Updates

## **Platform-UI - New user interface**
The new user interface has been developed from scratch, to provide the user a more native experience to federated learning.
The algorithm visualizations have also been reworked to provide a more meaningful view in the results.

## **Platform-Backend - Improvements to support the new user interface**
Other than required improvements to support the new user interface, the backend has also rewritten the communication
flow with the engine.

## **Exaflow – Federated Engine Revamp**

Exaflow, the former Exareme2 engine, underwent major improvements, allowing for the integration of other federated learning engines.

Closely integrated in exaflow is exareme3, our federated learning engine, using duckdb for data loading, grpc for 
communications and an aggregation server, allowing for a new paradigm in federated learning. 

Flower is also integrated, allowing for flower-developed algorithm integration as-is.

# Components

The main [MIP building blocks](documentation/Components.md) are listed along with the respective repositories that host them.

# Deployment

The MIP comes with a single code base but with two modes of deployment. One for local usage only, and one that enables the creation of a federation of nodes. Information on the different deployment approaches can be found in the following location:

- [Deployment Documentation](deployment)

# Federated Analysis Algorithms

This includes documentation on existing algorithm federation approach as well as information related to creating a new algorithm.

- [Available federated analysis algorithms](documentation/algorithms.md)
- [Exaflow Analytic Engine](https://github.com/madgik/exaflow/tree/1.0.0)

# Data Management

For all details relating to the Data Factory, how to manage your data and process it for use with the MIP, please consult the following document

- [Data Management Guide](documentation/MIP_Data_management_documentation.md)

A detailed user guide for Data Quality Control tool can be found here:
 - [Data Quality Control Tool Guide](https://github.com/HBPMedical/DataQualityControlTool/wiki)

Data Catalog is a component of the Medical Informatics Platform (MIP) for the EBrains. It enables seamless management, visualization, and access to data models and medical conditions.
 - [Data Catalog Guide](https://github.com/Medical-Informatics-Platform/datacatalog/tree/1.2.1)

# High Level Description

For a high-level description of the MIP please consult:

- [The MIP: A powerful federated data processing and analysis system that preserves patient privacy](https://ebrains.eu/data-tools-services/medical-analytics/medical-informatics-platform) on EBRAINS research infrastructure

# Architecture

- [High-level view of the architecture](documentation/Architecture.md), the main building blocks and data flows.

# Onboarding guidance

- [Onboarding to the Medical Informatics Platform MIP](https://wiki.ebrains.eu/bin/view/Collabs/onboarding-to-the-mip/) on EBRAINS Collaboratory

# Acknowledgement
This project/research received funding from the European Union’s Horizon 2020 Framework Programme for Research and Innovation under the Framework Partnership Agreement No. 650003 (HBP FPA).
