-- piko.query(name: GetConfig, command: one)
SELECT key_name, value FROM config WHERE key_name = ?;
