-- piko.name: InsertLog
-- piko.command: exec
INSERT INTO logs (id, message, created_at, unix_ts) VALUES (?, ?, ?, ?);

-- piko.name: GetLog
-- piko.command: one
SELECT id, message, created_at, unix_ts FROM logs WHERE id = ?;

-- piko.name: ListByDateRange
-- piko.command: many
SELECT id, message, created_at FROM logs WHERE created_at BETWEEN ? AND ? ORDER BY created_at ASC;

-- piko.name: FormatDate
-- piko.command: one
SELECT strftime('%Y', created_at) AS year, strftime('%m', created_at) AS month FROM logs WHERE id = ?;
