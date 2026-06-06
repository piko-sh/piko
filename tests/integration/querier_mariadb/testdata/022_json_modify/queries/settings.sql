-- piko.query(name: InsertSetting, command: exec)
INSERT INTO settings (name, config) VALUES (?, ?);

-- piko.query(name: GetSetting, command: one)
SELECT id, name, config FROM settings WHERE id = ?;

-- piko.query(name: SetConfigField, command: exec)
UPDATE settings SET config = JSON_SET(config, ?, ?) WHERE id = ?;

-- piko.query(name: ReplaceConfigField, command: exec)
UPDATE settings SET config = JSON_REPLACE(config, '$.theme', ?) WHERE id = ?;

-- piko.query(name: RemoveConfigField, command: exec)
UPDATE settings SET config = JSON_REMOVE(config, ?) WHERE id = ?;
