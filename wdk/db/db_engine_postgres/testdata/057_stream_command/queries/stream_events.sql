-- piko.name: StreamEvents
-- piko.command: stream
SELECT id, name, payload, created_at FROM events ORDER BY created_at;
