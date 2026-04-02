-- piko.name: UpsertEntry
-- piko.command: exec
INSERT INTO key_value_store (lookup_key, value) VALUES (?, ?)
ON DUPLICATE KEY UPDATE value = VALUES(value), version = version + 1;

-- piko.name: GetEntry
-- piko.command: one
SELECT id, lookup_key, value, version FROM key_value_store WHERE lookup_key = ?;

-- piko.name: ListEntries
-- piko.command: many
SELECT id, lookup_key, value, version FROM key_value_store ORDER BY lookup_key;
