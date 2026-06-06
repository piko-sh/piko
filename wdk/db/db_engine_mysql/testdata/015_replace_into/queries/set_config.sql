-- piko.query(name: SetConfig, command: exec)
REPLACE INTO config (key_name, value) VALUES (?, ?);
