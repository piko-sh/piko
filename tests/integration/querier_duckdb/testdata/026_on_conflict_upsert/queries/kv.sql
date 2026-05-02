-- piko.query(name: Upsert, command: exec)
INSERT INTO kv (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value, version = kv.version + 1;

-- piko.query(name: Get, command: one)
SELECT key, value, version FROM kv WHERE key = $1;
