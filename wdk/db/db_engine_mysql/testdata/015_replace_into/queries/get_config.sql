-- piko.name: GetConfig
-- piko.command: one
SELECT key_name, value FROM config WHERE key_name = ?;
