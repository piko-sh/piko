-- piko.name: CastValues
-- piko.command: many
SELECT
    id,
    CAST(raw_value AS DECIMAL(10, 2)) AS numeric_value,
    CAST(raw_value AS SIGNED) AS int_value
FROM measurements;
