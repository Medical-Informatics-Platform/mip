# CDEs JSON Changes for the Variable-Type Refactor

What changes **on the JSON side** of the `POST /excel-to-json` conversion under
the new MIP variable-types refactor. Companion to
[`EXCEL_REFACTOR_CHANGES.md`](./EXCEL_REFACTOR_CHANGES.md), which covers the
Excel side. Source: `converter/excel_to_json.py`, `validator/excel_validator.py`,
`common_entities.py`, and `EXCEL_TO_CDES_JSON_CROSSWALK.md`.

## Before and after the JSON fields

**Before** — the JSON carried three separate typing pieces, and `type` accepted
four values:

| `type` | `sql_type` | `is_categorical` |
| --- | --- | --- |
| `nominal` | `text` | `true` |
| `real` | `real` | `false` |
| `integer` | `int` | `false` |
| `text` | `text` | `false` |

**After** — `sql_type` and `is_categorical` are **no longer emitted**; `type`
carries the full classification, and `values` maps to `enumerations` or a range
depending on the type:

| `type` | Category | `values` → JSON field |
| --- | --- | --- |
| `continuous` | numerical | `minValue`, `maxValue` |
| `discrete` | numerical | `minValue`, `maxValue` |
| `nominal` | categorical | `enumerations` |
| `ordinal` | categorical | `enumerations` |
| `binary` | categorical | `enumerations` |

## What changed on the JSON output

- **`sql_type` and `isCategorical` are no longer emitted.** They used to be
  derived from `type`
  (`nominal → ("text", True)`, `real → ("real", False)`, `integer → ("int",
  False)`, `text → ("text", False)`); the refactor drops them entirely.
- **`type` carries the full classification.** The single `type` field is now the
  only typing information in the output, and it holds one of the five new values.
- **`values` maps to `enumerations` or `minValue`/`maxValue` by type.**
  Categorical types (`nominal`, `ordinal`, `binary`) produce `enumerations`;
  numerical types (`continuous`, `discrete`) produce a `minValue`/`maxValue`
  range. `discrete` bounds must be whole numbers.
- **`conceptPath` builds the group tree.** It is split on `/` to build the
  `groups` hierarchy, then removed from the variable. Example:
  `Group1/Subgroup/Var` → group `Group1` containing subgroup `Subgroup`
  containing variable `Var`.
- **Empty fields are stripped.** Empty fields and empty
  `groups`/`variables`/`enumerations` lists are removed from the output.

## Open question

> Some Excel `values` columns hold both enumeration values and min-max range
> values at the same time, so it is unclear whether the resulting JSON `values`
> field should carry `enumerations`, `minValue`/`maxValue`, or something else.
> See the open question in [`EXCEL_REFACTOR_CHANGES.md`](./EXCEL_REFACTOR_CHANGES.md).
