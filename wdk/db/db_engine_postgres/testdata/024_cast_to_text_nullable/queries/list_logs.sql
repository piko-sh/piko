-- piko.query(name: ListLogs, command: many)
SELECT id, level::TEXT AS level_text FROM logs;
