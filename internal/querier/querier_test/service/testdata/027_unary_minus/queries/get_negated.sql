-- piko.query(name: GetNegatedValues, command: many)
SELECT
  id,
  -value as negated_value,
  -offset_val as negated_offset
FROM measurements
