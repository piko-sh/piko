-- piko.name: UpsertSetting
-- piko.command: one
INSERT INTO settings (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
RETURNING key, value;
