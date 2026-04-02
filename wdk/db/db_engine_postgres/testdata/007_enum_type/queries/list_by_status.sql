-- piko.name: ListByStatus
-- piko.command: many
SELECT id, name, status FROM items WHERE status = $1::status;
