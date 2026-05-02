-- piko.query(name: InsertReading, command: exec)
INSERT INTO readings (ts, device_id, temperature) VALUES ($1, $2, $3);

-- piko.query(name: HourlyAverages, command: many)
SELECT
    time_bucket('1 hour'::interval, ts) AS bucket,
    device_id,
    avg(temperature) AS mean_temperature
FROM readings
WHERE device_id = $1
GROUP BY bucket, device_id
ORDER BY bucket;
