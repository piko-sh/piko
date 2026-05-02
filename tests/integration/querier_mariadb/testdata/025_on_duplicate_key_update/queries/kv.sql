-- piko.query(name: UpsertEntry, command: exec)
INSERT INTO key_value_store (lookup_key, value) VALUES (?, ?)
ON DUPLICATE KEY UPDATE value = VALUES(value), version = version + 1;

-- piko.query(name: GetEntry, command: one)
SELECT id, lookup_key, value, version FROM key_value_store WHERE lookup_key = ?;

-- piko.query(name: ListEntries, command: many)
SELECT id, lookup_key, value, version FROM key_value_store ORDER BY lookup_key;
