-- piko.query(name: ListTasks, command: many)
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;

-- piko.query(name: CreateTask, command: one)
INSERT INTO tasks (title, completed, created_at)
VALUES (?, 0, ?)
RETURNING id, title, completed, created_at;

-- piko.query(name: ToggleComplete, command: exec)
UPDATE tasks SET completed = CASE WHEN completed = 0 THEN 1 ELSE 0 END
WHERE id = ?;

-- piko.query(name: DeleteTask, command: exec)
DELETE FROM tasks WHERE id = ?;
