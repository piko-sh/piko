-- piko.name: GetEvent
-- piko.command: one
SELECT id, payload->>'name' AS event_name, payload->'metadata' AS meta FROM events WHERE id = $1;
