-- piko.name: GetProcessed
-- piko.command: many
SELECT id, nonexistent_func(name) AS processed FROM users WHERE id = ?
