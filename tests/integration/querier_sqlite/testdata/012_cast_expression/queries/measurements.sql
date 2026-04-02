-- piko.name: GetMeasurement
-- piko.command: one
SELECT
    id,
    CAST(raw_value AS REAL) AS numeric_value,
    CAST(raw_value AS INTEGER) AS integer_value,
    CAST(precision_value AS REAL) AS nullable_numeric
FROM measurements
WHERE id = ?;
