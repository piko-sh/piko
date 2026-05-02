-- piko.query(name: InsertGaugeSample, command: exec)
INSERT INTO gauge_samples (ts, value) VALUES ($1, $2);

-- piko.query(name: TimeWeightedAverage, command: one)
WITH summary AS (
    SELECT time_weight('LOCF', ts, value) AS state
    FROM gauge_samples
)
SELECT
    average(state)::double precision AS twa,
    first_val(state)::double precision AS first_value,
    last_val(state)::double precision  AS last_value
FROM summary;
