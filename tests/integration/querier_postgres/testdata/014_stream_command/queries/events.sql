-- piko.name: StreamEvents
-- piko.command: stream
SELECT id, name, timestamp FROM events ORDER BY id ASC;

-- piko.name: StreamEventsByName
-- piko.command: stream
SELECT id, name, timestamp FROM events WHERE name = $1 ORDER BY id ASC;
