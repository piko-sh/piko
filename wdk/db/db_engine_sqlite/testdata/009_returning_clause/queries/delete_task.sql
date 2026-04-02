-- piko.name: DeleteTask
-- piko.command: one
DELETE FROM tasks WHERE id = ? RETURNING id, title
