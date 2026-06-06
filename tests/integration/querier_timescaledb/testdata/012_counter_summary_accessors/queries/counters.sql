-- piko.query(name: InsertCounter, command: exec)
INSERT INTO counters (ts, value) VALUES ($1, $2);

-- piko.query(name: CounterAccessors, command: one)
WITH summary AS (
    SELECT counter_agg(ts, value) AS state
    FROM counters
)
SELECT
    slope(state)::double precision     AS slope_value,
    intercept(state)::double precision AS intercept_value,
    first_val(state)::double precision AS first_value,
    last_val(state)::double precision  AS last_value
FROM summary;
