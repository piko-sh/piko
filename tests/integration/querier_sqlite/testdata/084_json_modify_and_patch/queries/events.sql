-- piko.name: UpdateJsonField
-- piko.command: exec
UPDATE events SET data = json_set(data, '$.processed', ?) WHERE id = ?;

-- piko.name: RemoveJsonField
-- piko.command: exec
UPDATE events SET data = json_remove(data, '$.user.email') WHERE id = ?;

-- piko.name: GetEventData
-- piko.command: one
SELECT id, data FROM events WHERE id = ?;

-- piko.name: GetJsonType
-- piko.command: one
SELECT id, json_type(json_extract(data, '$.amount')) AS value_type FROM events WHERE id = ?;
