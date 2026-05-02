-- piko.query(name: InsertLog, command: exec)
INSERT INTO logs (id, message, created_at, unix_ts) VALUES (?, ?, ?, ?);

-- piko.query(name: GetLog, command: one)
SELECT id, message, created_at, unix_ts FROM logs WHERE id = ?;

-- piko.query(name: ListByDateRange, command: many)
SELECT id, message, created_at FROM logs WHERE created_at BETWEEN ? AND ? ORDER BY created_at ASC;

-- piko.query(name: FormatDate, command: one)
SELECT strftime('%Y', created_at) AS year, strftime('%m', created_at) AS month FROM logs WHERE id = ?;
