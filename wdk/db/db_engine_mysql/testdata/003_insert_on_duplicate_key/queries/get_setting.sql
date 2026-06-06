-- piko.query(name: GetSetting, command: one)
SELECT key_name, value, updated_at FROM settings WHERE key_name = ?;
