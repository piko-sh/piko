-- piko.query(name: InsertIntegerMetric, command: exec)
INSERT INTO integer_metrics (ts_epoch, sensor, value) VALUES ($1, $2, $3);

-- piko.query(name: PerMinuteSum, command: many)
SELECT
    time_bucket(60::bigint, ts_epoch) AS bucket,
    sum(value)                        AS total
FROM integer_metrics
GROUP BY bucket
ORDER BY bucket;
