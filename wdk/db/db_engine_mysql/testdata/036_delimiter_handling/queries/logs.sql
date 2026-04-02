-- piko.name: GetLogCount
-- piko.command: one
SELECT count_logs() AS total;

-- piko.name: ListLogs
-- piko.command: many
SELECT id, message, created_at FROM logs;
