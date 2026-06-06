-- piko.query(name: ListStores, command: many)
SELECT id, name, data FROM key_value_stores;

-- piko.query(name: GetMapKeys, command: many)
SELECT id, map_keys(data) AS keys FROM key_value_stores;

-- piko.query(name: GetMapValues, command: many)
SELECT id, map_values(data) AS vals FROM key_value_stores;
