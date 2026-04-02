-- piko.name: UpsertSetting
-- piko.command: one
-- $1 as piko.param(key)
-- $2 as piko.param(value)
INSERT INTO settings (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, updated_count = settings.updated_count + 1
RETURNING key, value, updated_count;
