-- piko.query(name: StreamEvents, command: stream)
SELECT id, name, timestamp FROM events ORDER BY id ASC;

-- piko.query(name: StreamEventsByName, command: stream)
SELECT id, name, timestamp FROM events WHERE name = ? ORDER BY id ASC;
