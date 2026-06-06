-- piko.query(name: GetSetting, command: one)
SELECT id, setting_key, value, description FROM settings WHERE setting_key = ?
