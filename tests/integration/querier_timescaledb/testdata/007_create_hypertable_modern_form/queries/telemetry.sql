-- piko.query(name: InsertTelemetry, command: exec)
INSERT INTO telemetry (ts, device_id, value) VALUES ($1, $2, $3);

-- piko.query(name: CountTelemetry, command: one)
SELECT count(*) FROM telemetry;
