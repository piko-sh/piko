-- piko.name: BulkInsertMeasurements
-- piko.command: copyfrom
INSERT INTO measurements (sensor_id, value, recorded_at) VALUES ($1, $2, $3);
