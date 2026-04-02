-- piko.name: UpsertKV
-- piko.command: one
INSERT INTO kv_store (key, value) VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, version = kv_store.version + 1
RETURNING key, value, version;
