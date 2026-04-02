-- piko.name: GetDoubledMeasurements
-- piko.command: many
SELECT id, double_value(value) AS doubled FROM measurements;

-- piko.name: GetLabelledMeasurements
-- piko.command: many
SELECT id, add_label(value) AS labelled FROM measurements;
