-- piko.query(name: CreateTask, command: one)
INSERT INTO tasks (title) VALUES (?) RETURNING id, title, done;

-- piko.query(name: MarkAllDone, command: many)
UPDATE tasks SET done = 1 WHERE done = 0 RETURNING id, title;

-- piko.query(name: DeleteTask, command: one)
DELETE FROM tasks WHERE id = ? RETURNING id, title;
