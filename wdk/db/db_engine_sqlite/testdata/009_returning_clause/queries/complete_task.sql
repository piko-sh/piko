-- piko.query(name: CompleteTask, command: one)
UPDATE tasks SET done = 1 WHERE id = ? RETURNING id, title, done
