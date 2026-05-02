-- piko.query(name: GetEventData, command: one)
SELECT id, CAST(data AS VARCHAR) AS data FROM events WHERE id = $1;

-- piko.query(name: GetJsonType, command: one)
SELECT typeof(json_extract(data, '$.amount')) AS value_type FROM events WHERE id = $1;

-- piko.query(name: ListJsonKeys, command: many)
SELECT id, CAST(json_keys(data) AS VARCHAR) AS keys FROM events ORDER BY id;
