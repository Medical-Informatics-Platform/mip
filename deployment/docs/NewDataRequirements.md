# Data Requirements

## Purpose and scope

This specification defines the required structure and content of data model
folders used for dataset onboarding. Each folder contains a `CDEsMetadata.json`
manifest and one or more CSV files. The manifest declares the type and meaning
of each variable; the CSV files contain the observation data.

The following requirements are fundamental:

* Each variable declares exactly one `type`.
* Categorical values are stored as integer codes. Human-readable values are
  declared as enumeration labels in the manifest.
* The `dataset` common data element (CDE) and its corresponding CSV column are
  mandatory.

For background on how variable types are interpreted by analysis algorithms,
see [Data Types, Storage and Variable Coding](https://github.com/madgik/exaflow/blob/8ebdc6f014479ad3d21d60e1072647b1a2809ae0/documentation/data-types-and-coding.md).

## Onboarding procedure

1. Create the data model folder and place `CDEsMetadata.json` and the CSV files
   in it.
2. Declare the `code`, `label` and `type` of every variable.
3. Encode categorical columns as integer codes and declare their labels in the
   manifest.
4. Validate the folder with the `exadata-validator` tool.
5. Load the validated folder.

## Folder structure

```text
/path/to/data_model_folder/
  CDEsMetadata.json
  dataset1.csv
  dataset2.csv
```

The folder name identifies the pathology.

## Reference example

The following metadata excerpt and CSV file illustrate a conforming data model.

`CDEsMetadata.json`:

```json
{
  "code": "dementia",
  "version": "0.1",
  "variables": [
    {
      "code": "dataset",
      "label": "Dataset",
      "type": "nominal",
      "enumerations": [
        { "code": "1", "label": "PPMI_0" },
        { "code": "2", "label": "EDSD_0" }
      ]
    }
  ],
  "groups": [
    {
      "code": "demographics",
      "label": "Demographics",
      "variables": [
        { "code": "subjectcode", "label": "Subject code", "type": "discrete" },
        {
          "code": "gender",
          "label": "Gender",
          "type": "binary",
          "enumerations": [
            { "code": "0", "label": "Male" },
            { "code": "1", "label": "Female" }
          ]
        },
        {
          "code": "diagnosis",
          "label": "Diagnosis",
          "type": "nominal",
          "enumerations": [
            { "code": "1", "label": "Healthy control" },
            { "code": "2", "label": "MCI" },
            { "code": "3", "label": "Alzheimer's disease" }
          ]
        },
        { "code": "age", "label": "Age", "type": "continuous", "minValue": 0, "maxValue": 120 }
      ]
    }
  ]
}
```

`dataset1.csv`:

```csv
dataset,subjectcode,gender,diagnosis,age
1,1001,1,2,72.4
1,1002,0,1,68.1
2,2001,1,3,80.0
```

Key observations:

* `gender` is declared `binary`. The CSV stores `1` or `0`, which the manifest
  maps to "Female" and "Male" respectively.
* `diagnosis` is declared `nominal`. The CSV stores `2`, which the manifest maps
  to "MCI".
* `age` is declared `continuous`; the CSV stores a numeric value such as `72.4`.
* `dataset` is mandatory. The CSV stores `1` or `2`, corresponding to the labels
  `PPMI_0` and `EDSD_0`.

## Metadata specification

The manifest is a tree structure in which `groups` are branches and `variables`
are leaves. Groups may contain further groups. The `code` of each variable is
the name of the corresponding CSV column.

### Required variable attributes

| Attribute | Definition |
| --- | --- |
| `code` | The CSV column name. Must be a valid Python identifier and unique within the data model. |
| `label` | The human-readable name of the variable. |
| `type` | One of the five variable types defined below. |

### Variable types

* **`continuous`** — a numeric value that may take any value in a range, such as
  `72.4`. Stored as `DOUBLE`.
* **`discrete`** — a whole number, such as `42`. Stored as `INTEGER`.
* **`binary`** — a variable with exactly two categories. Declare exactly two
  enumerations with codes `0` and `1`, in that order. Code `0` is the reference
  level.
* **`nominal`** — unordered categories. Declare enumerations with integer codes
  starting at 1, normally `1..K`.
* **`ordinal`** — ordered categories. Codes follow the `nominal` convention, and
  the declaration order defines the rank.

Each categorical level is declared as a `code`/`label` pair:

```json
{ "code": "1", "label": "Female" }
```

The code is the value stored in the CSV; the label is the human-readable value.
Values not declared in the enumerations are rejected during validation and
loading. Where a data model is deployed at multiple sites, all sites must
declare the same levels in the same order.

### Optional attributes

* **`minValue` and `maxValue`** — applicable to `continuous` and `discrete`
  variables only. Either or both may be supplied; when both are supplied,
  `minValue` must be less than `maxValue`. For `discrete` variables, bounds must
  be whole numbers. The aliases `min` and `max` are also accepted. Bounds must
  not be declared for categorical variables.
* **`description`, `units`, `methodology`** — optional descriptive attributes.

### Mandatory and reserved variables

* **`dataset`** is mandatory. It must declare `"type": "nominal"` and
  enumerations with integer codes `1..N`. Each label (for example `PPMI_0`)
  must be unique within the data model, as requests identify datasets by label.
  The CSV stores the codes rather than the labels.
* **`subjectcode` and `subjectid`** are, by convention, `discrete` whole numbers
  without enumerations, so that subject identifiers are not exposed as category
  levels. This convention is not enforced by the validator.

The parent object must also declare a `version` attribute of type string.

## CSV file specification

* The first row is a header containing variable `code`s. Each subsequent row is
  one observation.
* Every column must be declared in `CDEsMetadata.json`. Columns may be omitted
  relative to the manifest, but additional columns are not permitted; `row_id`
  is the sole internal exception.
* The `dataset` column is mandatory in every CSV and must contain the integer
  codes declared in the `dataset` enumerations.
* Categorical columns must contain integer codes, never human-readable labels:
  use `1` rather than `PPMI_0`, `1` rather than `Female`, and `1` rather than
  `BL`.
* Codes must be written as plain integers. Values such as `1.0` and `1.5` are
  invalid.
* `continuous` columns contain finite numeric values; `discrete` columns contain
  whole numbers.

## Longitudinal data models

For data with repeated visits, set `"longitudinal": true` at the top level of
`CDEsMetadata.json`, alongside `version`.

Metadata requirements:

* A `subjectid` CDE is mandatory.
* A `visitid` CDE is mandatory. It must declare a categorical `type` with
  enumerations using integer codes `1..K`. Visit names such as `BL`, `FL1` and
  `FL2` are enumeration labels, not stored values.

CSV requirements:

* `subjectid` and `visitid` must be present and non-null on every row.
* No two rows may share the same `(subjectid, visitid)` pair.

## Validation

Install and execute the validator against the folder:

```bash
pip install exadata-validator
exadata-validator validate-data-model /path/to/data_model_folder
```

Validation covers variable types, integer category codes, enumerations, bounds,
required columns, dataset codes and longitudinal constraints. Run the command
with `--help` for reporting and output options.

## Loading data

* **New pathology** — create a folder in the `data` directory named after the
  pathology and place `CDEsMetadata.json` and the CSV files in it.
* **Additional data for an existing pathology** — add the CSV files to the
  existing pathology folder, using the declared codes and leaving the manifest
  unchanged.

## Migrating legacy data

The legacy attributes `isCategorical` and `sql_type` are ignored by the
validator and should be removed. Migrate legacy `type` values as follows:

| Legacy `type` | Current `type` |
| --- | --- |
| `real` | `continuous` |
| `integer` | `discrete` |
| `nominal` | `nominal` |
| `text` | `nominal`, or `ordinal` when the categories are ordered |
| — | `binary` for variables with exactly two categories |

Recode text values in the CSV files to integer codes and transfer the readable
values to the enumeration labels. Use codes `1..N` for `dataset` and `1..K` for
`visitid`.

Previously deployed data is not migrated automatically. The integer-code
storage convention applies only to newly loaded data.
