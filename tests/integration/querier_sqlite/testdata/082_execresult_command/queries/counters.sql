-- piko.name: IncrementCounter
-- piko.command: execresult
UPDATE counters SET value = value + ? WHERE name = ?;

-- piko.name: GetCounter
-- piko.command: one
SELECT id, name, value FROM counters WHERE name = ?;
