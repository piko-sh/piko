-- piko.query(name: StreamEvents, command: stream)
SELECT id, name, payload, created_at FROM events ORDER BY created_at;
