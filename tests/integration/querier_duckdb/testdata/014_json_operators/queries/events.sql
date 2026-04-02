-- piko.name: GetEventName
-- piko.command: one
SELECT id, json_extract_string(data, '$.name') AS event_name FROM events WHERE id = $1;

-- piko.name: GetNestedValue
-- piko.command: one
SELECT id, data->>'$.user.email' AS email FROM events WHERE id = $1;

-- piko.name: ListByCategory
-- piko.command: many
SELECT id, name, data->>'$.category' AS category FROM events WHERE data->>'$.category' = $1 ORDER BY id;
