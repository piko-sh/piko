-- piko.name: ListStores
-- piko.command: many
SELECT id, name, data FROM key_value_stores;

-- piko.name: GetMapKeys
-- piko.command: many
SELECT id, map_keys(data) AS keys FROM key_value_stores;

-- piko.name: GetMapValues
-- piko.command: many
SELECT id, map_values(data) AS vals FROM key_value_stores;
