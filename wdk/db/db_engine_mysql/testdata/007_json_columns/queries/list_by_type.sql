-- piko.name: ListByType
-- piko.command: many
SELECT id, event_type, payload->>'$.name' AS event_name
FROM events
WHERE event_type = ?;
