-- piko.name: RecentLogs
-- piko.command: many
SELECT id, message, level, created_at FROM logs ORDER BY created_at DESC LIMIT ?;
