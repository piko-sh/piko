-- piko.query(name: InsertSparseReading, command: exec)
INSERT INTO sparse_readings (ts, sensor_id, value) VALUES ($1, $2, $3);

-- piko.query(name: GapfilledHourlyAverages, command: many)
SELECT
    time_bucket_gapfill('1 hour'::interval, ts, $2::timestamptz, $3::timestamptz) AS bucket,
    locf(avg(value)) AS filled_value
FROM sparse_readings
WHERE sensor_id = $1
  AND ts >= $2::timestamptz
  AND ts <  $3::timestamptz
GROUP BY bucket
ORDER BY bucket;
