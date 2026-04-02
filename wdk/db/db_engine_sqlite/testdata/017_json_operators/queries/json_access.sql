-- piko.name: GetEventType
-- piko.command: one
SELECT id, payload -> '$.type' AS event_type, payload ->> '$.name' AS event_name FROM events WHERE payload ->> '$.active' = ?
