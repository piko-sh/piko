-- piko.name: InsertSetting
-- piko.command: exec
INSERT INTO settings (name, config) VALUES (?, ?);

-- piko.name: GetSetting
-- piko.command: one
SELECT id, name, config FROM settings WHERE id = ?;

-- piko.name: SetConfigField
-- piko.command: exec
UPDATE settings SET config = JSON_SET(config, ?, ?) WHERE id = ?;

-- piko.name: ReplaceConfigField
-- piko.command: exec
UPDATE settings SET config = JSON_REPLACE(config, '$.theme', ?) WHERE id = ?;

-- piko.name: RemoveConfigField
-- piko.command: exec
UPDATE settings SET config = JSON_REMOVE(config, ?) WHERE id = ?;
