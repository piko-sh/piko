-- piko.query(name: InsertSample, command: exec)
INSERT INTO samples (ts, value) VALUES ($1, $2);

-- piko.query(name: BottomThree, command: one)
WITH agg AS (
    SELECT min_n(value, 3) AS state
    FROM samples
)
SELECT (into_array(state))::text AS lowest_values_text
FROM agg;
