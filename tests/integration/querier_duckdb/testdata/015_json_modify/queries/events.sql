-- piko.name: GetEventData
-- piko.command: one
SELECT id, CAST(data AS VARCHAR) AS data FROM events WHERE id = $1;

-- piko.name: GetJsonType
-- piko.command: one
SELECT typeof(json_extract(data, '$.amount')) AS value_type FROM events WHERE id = $1;

-- piko.name: ListJsonKeys
-- piko.command: many
SELECT id, CAST(json_keys(data) AS VARCHAR) AS keys FROM events ORDER BY id;
