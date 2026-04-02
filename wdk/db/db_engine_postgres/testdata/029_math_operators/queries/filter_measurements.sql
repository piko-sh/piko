-- piko.name: FilterMeasurements
-- piko.command: many
SELECT id, value FROM measurements WHERE value >= $1 AND value <= $2 AND (value * $3) > $4;
