-- piko.query(name: NextCounterValue, command: one)
SELECT nextval('counters_seq') AS next_id;

-- piko.query(name: CountRows, command: one)
SELECT COUNT(*) AS total FROM counters;
