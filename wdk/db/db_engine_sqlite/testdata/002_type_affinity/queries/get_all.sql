-- piko.name: GetAllTypedValues
-- piko.command: many
SELECT
  id, int_val, bigint_val, text_val, varchar_val,
  real_val, double_val, blob_val, numeric_val,
  bool_val, date_val, datetime_val, json_val
FROM typed_values
