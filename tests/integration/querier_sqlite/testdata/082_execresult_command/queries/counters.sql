-- piko.query(name: IncrementCounter, command: execresult)
UPDATE counters SET value = value + ? WHERE name = ?;

-- piko.query(name: GetCounter, command: one)
SELECT id, name, value FROM counters WHERE name = ?;
