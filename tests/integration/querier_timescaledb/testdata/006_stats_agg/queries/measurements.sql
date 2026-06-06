-- piko.query(name: InsertMeasurement, command: exec)
INSERT INTO measurements (ts, group_id, sample) VALUES ($1, $2, $3);

-- piko.query(name: StatsByGroup, command: many)
WITH summaries AS (
    SELECT group_id, stats_agg(sample) AS summary
    FROM measurements
    GROUP BY group_id
)
SELECT
    group_id,
    num_vals(summary)::bigint AS sample_count,
    average(summary)::double precision AS mean_value,
    stddev(summary)::double precision AS stddev_value
FROM summaries
ORDER BY group_id;
