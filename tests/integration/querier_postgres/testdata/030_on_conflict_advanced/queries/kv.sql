-- piko.name: ConditionalUpsert
-- piko.command: one
INSERT INTO kv (key, value)
VALUES ($1, $2)
ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, version = kv.version + 1
WHERE kv.version < 10
RETURNING key, value, version;

-- piko.name: GetKeyValue
-- piko.command: one
SELECT key, value, version
FROM kv
WHERE key = $1;
