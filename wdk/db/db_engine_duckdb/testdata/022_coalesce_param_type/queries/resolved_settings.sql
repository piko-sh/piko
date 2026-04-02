-- piko.name: ResolvedSettings
-- piko.command: many
SELECT id, COALESCE(value, $1) AS resolved_value FROM settings;
