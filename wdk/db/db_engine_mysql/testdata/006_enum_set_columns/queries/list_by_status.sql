-- piko.query(name: ListByStatus, command: many)
SELECT id, title, status FROM articles WHERE status = ?;
