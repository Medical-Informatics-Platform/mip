# Excel File Changes for the Variable-Type Refactor

What must change **in the Excel file** for the new MIP variable-types refactor,
for the `POST /excel-to-json` conversion. Companion to
[`JSON_REFACTOR_CHANGES.md`](./JSON_REFACTOR_CHANGES.md), which covers the JSON
side. Source: `converter/excel_to_json.py`, `validator/excel_validator.py`,
`common_entities.py`, and `EXCEL_TO_CDES_JSON.md`.

## 1. `type` values: four → five

The `type` column is the single gate for how a variable is stored. It no longer
accepts the old four values.

**Before** — accepted `nominal`, `real`, `integer`, `text`:

| `type` | old `sql_type` | old `is_categorical` |
| --- | --- | --- |
| `nominal` | `text` | `true` |
| `real` | `real` | `false` |
| `integer` | `int` | `false` |
| `text` | `text` | `false` |

**After** — accepts **only** `continuous`, `discrete`, `nominal`, `ordinal`,
`binary`:

| `type` | Category | Definition |
| --- | --- | --- |
| `continuous` | numerical | Numeric values that can take any value in a range (e.g., blood pressure, lab values) |
| `discrete` | numerical | Numeric values that take distinct steps (e.g., age in years, number-like integers) |
| `nominal` | categorical | Unordered categories (e.g., blood type, sex) |
| `ordinal` | categorical | Ordered categories (e.g., cancer stage, Likert scale) |
| `binary` | categorical | Two-category variable (e.g., yes/no, positive/negative) |

Any other value in `type` is rejected with a `400`.

### Old → new type mapping

How to migrate an existing Excel file from the old four values to the new five:

| Old `type` | New `type` | Notes |
| --- | --- | --- |
| `real` | `continuous` | numeric in a range |
| `integer` | `discrete` | numeric in distinct steps |
| `nominal` | `nominal` | unordered categories |
| `text` | `nominal` or `ordinal` | pick `ordinal` if the categories are ordered, otherwise `nominal` |
| — (new) | `binary` | two-category variables; no old equivalent |

## 2. `sql_type` and `is_categorical` columns

`sql_type` and `is_categorical` are **dropped entirely** from the output.

- If your Excel file has `sql_type` or `is_categorical` columns, they are not in
  the accepted header set
  (`csvFile`, `name`, `code`, `type`, `values`, `unit`, `description`,
  `canBeNull`, `comments`, `conceptPath`, `methodology`) and the workbook is
  rejected with a `400` — remove them.
- You no longer need to supply them: the `type` value alone carries the full
  classification.

## 3. `values` column by type

The `values` column format is driven by the new `type`:

| `type` | `values` column format |
| --- | --- |
| `nominal` / `ordinal` / `binary` | `"{"code", "label"}, ..."` pairs; codes must be unique |
| `continuous` | `<min>-<max>` range; `min` < `max` |
| `discrete` | `<min>-<max>` range; `min` < `max`; whole numbers only |

## Open question

> Some Excel columns in the `values` field appear to hold **both** enumeration
> values (`{"code", "label"}` pairs) **and** min-max range values at the same
> time. It is unclear how such a column should be interpreted — as
> enumerations, as a range, or something else — and this needs to be decided and
> handled explicitly before the refactor lands.
