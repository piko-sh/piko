-- piko.name: ListLogs
-- piko.command: many
SELECT id, level::TEXT AS level_text FROM logs;
