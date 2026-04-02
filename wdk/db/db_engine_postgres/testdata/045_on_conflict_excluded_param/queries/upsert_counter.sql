-- piko.name: UpsertCounter
-- piko.command: one
INSERT INTO counters (key, count, label) VALUES ($1, $2, $3)
ON CONFLICT (key) DO UPDATE SET count = counters.count + EXCLUDED.count, label = $3
RETURNING key, count, label;
