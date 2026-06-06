-- piko.query(name: GetReadings, command: many)
SELECT id, COALESCE(sensor_a, sensor_b) AS best_reading FROM readings ORDER BY id;
