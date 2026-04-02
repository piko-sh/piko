-- piko.name: CompleteTask
-- piko.command: one
UPDATE tasks SET done = 1 WHERE id = ? RETURNING id, title, done
