-- piko.query(name: UpsertSetting, command: exec)
INSERT INTO settings (key_name, value)
VALUES (?, ?)
ON DUPLICATE KEY UPDATE value = VALUES(value);
