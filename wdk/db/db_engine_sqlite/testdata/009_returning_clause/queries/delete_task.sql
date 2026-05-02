-- piko.query(name: DeleteTask, command: one)
DELETE FROM tasks WHERE id = ? RETURNING id, title
