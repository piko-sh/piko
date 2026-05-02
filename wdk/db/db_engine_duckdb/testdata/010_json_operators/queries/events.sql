-- piko.query(name: GetEventName, command: many)
SELECT id, payload->>'name' AS event_name FROM events;

-- piko.query(name: GetEventMetadata, command: many)
SELECT id, payload->'metadata' AS metadata FROM events;
