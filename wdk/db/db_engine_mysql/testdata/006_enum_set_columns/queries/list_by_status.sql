-- piko.name: ListByStatus
-- piko.command: many
SELECT id, title, status FROM articles WHERE status = ?;
