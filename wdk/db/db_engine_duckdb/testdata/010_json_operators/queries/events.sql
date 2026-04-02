-- piko.name: GetEventName
-- piko.command: many
SELECT id, payload->>'name' AS event_name FROM events;

-- piko.name: GetEventMetadata
-- piko.command: many
SELECT id, payload->'metadata' AS metadata FROM events;
