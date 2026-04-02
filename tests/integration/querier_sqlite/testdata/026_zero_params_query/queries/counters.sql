-- piko.name: CountAll
-- piko.command: one
SELECT count(*) AS total FROM counters;

-- piko.name: ListCounters
-- piko.command: many
SELECT id, name, value FROM counters ORDER BY id;
