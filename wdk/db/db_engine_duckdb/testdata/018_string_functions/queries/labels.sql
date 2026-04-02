-- piko.name: GetTransformedLabels
-- piko.command: many
SELECT id, upper(value) AS upper_val, lower(value) AS lower_val, length(value) AS len FROM labels;
