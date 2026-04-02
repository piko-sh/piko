-- piko.name: CreateTask
-- piko.command: one
INSERT INTO tasks (title) VALUES (?) RETURNING id, title, done;

-- piko.name: MarkAllDone
-- piko.command: many
UPDATE tasks SET done = 1 WHERE done = 0 RETURNING id, title;

-- piko.name: DeleteTask
-- piko.command: one
DELETE FROM tasks WHERE id = ? RETURNING id, title;
