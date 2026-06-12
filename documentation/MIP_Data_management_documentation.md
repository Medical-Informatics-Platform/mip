# **Data Management Guideline for the Medical Informatics Platform (MIP)**

# **Table of Contents**

* [About this Document](#about-this-document)

  * [Target Audience](#target-audience)

  * [Document Overview](#document-overview)

  * [Concepts and Definitions](#concepts-and-definitions)

* [Data Management and Processing](#data-management-and-processing)

  * [Extraction and compilation of clinical data as a csv file](#extraction-and-compilation-of-clinical-data-as-a-csv-file)

  * [Definition of Common Data Elements for a medical condition](#definition-of-common-data-elements-for-a-medical-condition)

  * [Data cleaning, validation, and upload](#data-cleaning-validation-and-upload)

# About this Document

## Target Audience

This document is targeting data processors and members of data controllers from institutes interested in analysing their clinical and research data within the MIP. It mainly addresses technical coordinators and data managers who want to prepare their data to be ingested into the MIP analysis engine.

The document provides the information needed to understand:

- The approach of data processing to achieve data ingestion to the MIP.
- The tools that have been developed to facilitate the data processing (The MIP Data Catalogue, and the MIP Data Quality Control Tool (MIP-DQC Tool).
- Link to the user guide for each of these tools

## Document Overview

This document describes the MIP Data architecture, and for which purpose it was assembled. It explains the overall data processing required to conform to MIP standards before being uploaded to the MIP and how the MIP Data Quality tools shall be used to help achieving these standards.

It is a step-by-step guide covering data end to end processing and it provides links to additional documentation and detailed instructions described in the different GitHub repositories and shall not be seen in isolation.

## Concepts and Definitions

**Common Data Elements (CDEs)**

A set of standard variables defined by clinical experts and data scientists, which would be used by researchers to perform analysis on specific medical conditions at the federation level. In the MIP context, we use the term CDEs to refer to the standardized federated data models only.

**Data Element**

In metadata, the term data element is an atomic unit of data that has precise meaning or precise semantics. [Beynon-Davies P. (2004). Database Systems 3rd Edition. Palgrave, Basingstoke, UK]

**Data Models (Metadata)**

Data Model (Metadata) describes the structure of database variables found in specific extract of a hospital's database, including descriptive metadata, structural metadata, administrative metadata, reference metadata and statistical metadata. [1] Zeng, Marcia (2004). "Metadata Types and Functions". NISO. Archived from the original on 7 October 2016. Retrieved 5 October 2016.

**Database Variables:**

A variable or scalar is a storage address (identified by an index or address) paired with an associated symbolic name, which contains some known or unknown quantity of information referred to as a value. [Knuth, Donald (1997). The Art of Computer Programming. 1 (3rd ed.). Reading, Massachusetts: Addison-Wesley. p. 3-4.]

**Electronic Health Records (EHR)**

Health information and clinical records registered per each patient per visit in the hospital's database (Oracle, SQL or any database systems) and usually transferred in db or CSV format. EHR usually contain different levels of data; we might define them in this context as spaces, domain, and sub-domain. For example, General space might include demographic, social status or patient's medical history as different data domains. On the other hand, EHR contain other data spaces related to the specific medical condition such as Dementia or Epilepsy where each space includes specific domain and sub-domain, such as medical assessments and tests, diagnoses, treatment, and operations, etc.

# Data Management and Processing

Data originating from diverse hospitals are highly heterogeneous in nature and hence cannot be uploaded into the MIP _per se_. Multiple steps need to be done that are represented in the following figure and which can be regrouped in two main groups:

- Identification of variables that will be made available in the MIP to fulfil the scientific analysis objective and definition of the Data Model (Metadata). In case of a federated MIP, all hospitals must agree to a Common Data Elements (CDEs).
- Data pre-processing to conform to the CDEs predefined and to the MIP's standard formats.

The standard format of the dataset file is .csv UTF-8. Because csv format doesn't allow to keep the Metadata structure, a second file is required, which corresponds to the CDEs for which the MIP standard format is JSON.

To help data managers to go through these processes, two main tools have been developed until now:

- **The MIP Data Catalogue** : Web-based application that is used to explore and manage MIP CDEs pre-defined by the federation's community or the use case group. All the existing Medical Conditions, Federations and their associated Common Data Elements (CDEs) are available within the MIP Data Catalogue, which is the unique source of reference for the MIP and the MIP-DQC Tool. The MIP Data Catalogue can be accessed at the following address ([HERE](https://datacatalogue.mip.ebrains.eu/#federations)).
- **The MIP Data Quality Control Tool (MIP-DQC Tool)**: A standalone software that provides hospital personnel an easy way to explore, transform and validate their dataset based on the CDEs before uploading them into the MIP. MIP-DQC Tool has both, a Command Line Interface (CLI) and a Graphical User Interface (GUI), with only the latter having the full set of functionalities. The MIP-DQC Tool GUI version has the following main functionalities:
  - Inference of a dataset's schema and producing a schema file in Frictionless json or the MIP Data Catalogue's excel format.
  - Validating the MIP Data Models or CDEs pre-defined by the federation's community or the use case group.
  - Translating the MIP Data Models or CDEs from xlsx format to json format.
  - Validating the hospital tabular (csv) data against the MIP Data Models or CDEs and producing a validation report and some overall statistics about the data.
  - Data cleaning capability based on the previously performed validation report.

The user guide for each of these three tools can be found in the respective GitHub repositories. 
  - [The MIP Data Catalogue](https://datacatalogue.mip.ebrains.eu/#federations)
  - [The MIP Data Quality Control Tool](https://github.com/HBPMedical/DataQualityControlTool/wiki)

Through description of the steps required before being able to upload data on the MIP, the information of which tool can be used is provided in italics. The following table lists the sequential steps as well as the tool that can be used to simplify this process.

***Table 1*: MIP Data extraction and compilation and available Tools**
|| **1srt step** | **2nd step (optional)** | **3rd step (optional)** |
| --- | --- | --- | --- |
| **Tasks** | Extract the clinical variable as a csv file | Extract relevant variable from medical images | Merge those data to obtain one csv file |
| **Tools** | | | |
 
***Table 2*: MIP Common Data Element definition and available Tools**
|| **1srt step** | **2nd step** | **3rd step** |
| --- | --- | --- | --- |
| **Tasks** |Infer the data model from the csv file to produce the Data Model xlsx file | For the federation, there should be a discussion around the standardization of variables in the Data Model to produce a standard Common Data Element (CDE) as an xlsx file | Validate and translate the CDEs xlsx file to JSON format |
| **Tools** | MIP-DQC Tool | | MIP-DQC Tool & MIP Data Catalogue |


***Table 3*: MIP Data cleaning, validation, and upload and available Tools**
|| **1srt step** | **2nd step** | **3rd step** |
| --- | --- | --- | --- |
| **Tasks** | Check the dataset csv file quality against the CDEs | Perform the data cleaning and validation | Upload of the dataset csv file on the local node |
| **Tools** | MIP-DQC Tool | MIP-DQC Tool |

![The MIP data flow](./images/The%20MIP%20Data%20flow.jpg)

## Extraction and compilation of clinical data as a csv file

### Extract variables from the Electronic Health Record (EHR) system

The dataset that can be uploaded on the local node is a unique csv file that contains all the variables. The first step is hence to extract the clinical data of interest from the EHR system to a csv file. In most cases, a hospital EHR system contains various examination results that a patient obtains during one or more hospital visits. 

In case of longitudinal dataset, the csv file must contain:

- a field with a unique ID for each PATIENT: `subjectID`
- a field with a unique ID for each VISIT (BL, FL1, FL2...): `visitID`
- a "Number of month since first visit" field for each VISIT: `timeBL`

In terms of csv file structure, the following basic conditions must be met:

- Header that contains the variable names and must:
  - not contain special characters like space, parentheses, hyphen, etc.
  - use only underscore ("\_") for word separation.
  - not start with any number character.
- The csv delimiter must be the comma character (",").
- The encoding must be ASCII.

### Extract Medical Images Features - OPTIONAL

Not all medical conditions require data extracted from medical images which explains why this step can be skipped. However, because the MIP recognizes only one csv data format, features from medical images must be extracted and merged to the csv file that contains EHR variables in case they are valuable for the scientific analysis objectives.

Although multiple methodologies can be used to extract medical images features, we highly recommend following the same extraction pipeline to harmonize the methodology across the hospitals.

As soon as all variables are merged in a unique csv file, you can proceed with the definition of the Data Model and the Common Data Elements for your medical condition and your scientific analysis objectives.

## Definition of Common Data Elements for a medical condition

### Data Model (Metadata) xlsx File format

Hospitals that want to upload a dataset on the MIP need first to create the Data Model (Metadata) of its dataset.

The Data Model xlsx file contains information about the hospital variables. It corresponds to a table that lists the variables present in the hospital data set and describes them. A template can be downloaded from the Data Catalogue web-based application.

The descriptions required are listed below; in **bold** the ones that cannot be left empty:

- **name** : The full name of the variable that will appear on the MIP interface
- **code** : The code version of the variable name that is used on the csv file of the dataset
- **type** : Format of the variable. Five formats are recognized in the MIP:
  - Categorical variable:
    - Nominal - multiple level without hierarchy (Handedness for exemple)
    - Ordinal - multiple level with a possible hierarchy (graduate stage of the disease for exemple, be carefull that the difference between each steps are similar) 
    - Binary - two level without hierarchy (Presence/Absence variable for exemple)
  - Numerical:
    - Integer - Discrete (Age in year for exemple)
    - Real - Continuous (Glucose concentration for exemple)
- **values** : Only for the categorical variable. It lists all the levels of the categorical variable with the correspondence between the code and the value. The format used is - {"Code","Value"},{"Code","Value"}. For example, the sex variable is coded as: {"1","Male"},{"2","Female"}. Note that to increase the analitic the code should be integer.
- unit: Only for numerical variables. Unit of the variable (_e.g.,_ km/h)
- canBeNull: Mention with yes/no the possibility of this variable to be null. It is mainly useful for MIP federated nodes.
- description: A precise description of the variable.
- comments: Any comment. For example, a linkage with another variable.
- **conceptPath** : Hierarchy of the variable within the dataset as MedicalCondition/Section/Sub-section/Sub-sub-section etc… This allows the MIP-Front end to create hierarchy visualization of the data. There is no specific number of hierarchy levels required and can be customized to a specific Data Model.
- methodology: Specification of the methodology followed to obtain this variable when different methodologies exist.

!NOTE : Although the semantics and the conceptPath can be customized to a specific Data Model, we highly recommend following existing Standards such as HL7 FHIR, SNOMED or ICD-10

*The **Data Quality Control Tool** allows you to infer this Data Model xlsx file based on your data csv file. In the infer option section of the Data Quality Control Tool, indicate the number of rows that the tool will be based on for the schema inference and the maximum number of categories that a nominal variable can have.*

### Common Data Elements

Based on the Data Model, a hospital selects the variables that will be made available on the MIP by keeping only the selected variables in the Data Model xlsx file. The CDE file must be named as: "Medical Condition"\_cdes\_"version" (e.g. "dementia\_cdes\_v6").

In case of a MIP-Federated node, three more steps are required:

- Each variable that could lead to identification of a patient must be deleted (i.e., name, date of birth, address…) and must be replaced by a patient code. 
- Each hospital that is part of the federation needs to agree on the list of the variables AND on the harmonization of each variable. Indeed, variables from each hospital need to have the exact same codification and meaning for nominal variables and, should have been acquired with a similar methodology to be federated. Here is a list of discussion points:
  - Agreement on the Variable list
  - For each categorical variable, agreement of codification and meaning of this codification
  - For each numerical variable, agreement of the unit and range
  - For each variable, agreement of the methodology followed to acquire it

- Add a nominal variable named `Dataset` that have a level for each hospital federated. A code for each hospital needs to be created (e.g. Centre Hospitalier Universitaire Vaudois is coded `1` and is written as {"1"," Centre Hospitalier Universitaire Vaudois "}

As soon as the CDE is defined, and in the same xlsx format as the Data Model, it can be translated to the json format that is the standard format of the MIP. To do so, you can use the Data Quality Control Tool to have first a validation control to avoid any mistake and second translate it to json format.

*The **Data Quality Control Tool** allows you to validate your Data Model or CDEs to avoid missing cells, extra space etc… . In the Validate DC Excel section of the Data Quality Control Tool, upload your Data Model or CDEs in xlsx format and click on Validate. In case there is no mistake, you can translate it automatically to json format. This translation is also made by **the Data Catalogue** web-based application.*

Once the CDEs have been defined, the process of preparing hospital data for ingestion to the MIP can be started.

!NOTE: In case a hospital wants to join an existing federation, it should download the pre-defined CDEs from the Data Catalogue web-based application and contact the MIP team support to be able to join the federation.

## Data cleaning, validation, and upload

### Check of the dataset quality against the CDEs

Based on the CDEs defined is the previous steps, the dataset, compiled as a csv, needs to be cleaned/modified and validated according to the definition of all variables. All the column names that exist in the csvs **must** also exist in the CDEsMetadata.json file. The two main violation types that need to be fixed are:
- Constraint violations which regroup:
  - minimum (for numerical variable)
  - maximum (for numerical variable)
  - enum (list of enumerations for categorical variable)
- Datatype violations: Case when a value in a column has a different datatype than the one that has been declared for that variable in the CDEs. This also includes the empty cells that should be truly empty and not with "NA", "Nas", "unknowns" etc…

### Perform the data cleaning and validation

All violations highlighted before should be corrected so that each variable present in the csv file, corresponds to the one listed in the CDEs. The csv can have less columns than the CDEs in the metadata but **NOT** more. Be careful to save the csv file in the correct format, which is csv; encoding must be ASCII with comma separation.

### Data set upload

To become visible to the MIP end users, the dataset csv must be first renamed as the hospital code defined on the Dataset variable created on the CDE (e.g., "chuv.csv"). The file is then copied to the corresponding "medical condition" folder on the MIP local node and, when the hospital is part of a federation (federated hospital), to the corresponding "medical condition" folder on the MIP federated node.

*The **Data Quality Control Tool** allows you to compare the CDEs json file with your dataset csv file to highlight the violations and suggests possible corrections. If the corrections suggestions are good, the DQC tool can also perform those corrections and save the corrected dataset in the correct format.*