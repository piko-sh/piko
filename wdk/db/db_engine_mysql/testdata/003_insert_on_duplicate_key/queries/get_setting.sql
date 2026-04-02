-- piko.name: GetSetting
-- piko.command: one
SELECT key_name, value, updated_at FROM settings WHERE key_name = ?;
