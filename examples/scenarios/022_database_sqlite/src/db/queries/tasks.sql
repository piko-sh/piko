-- piko.name: ListTasks
-- piko.command: many
SELECT id, title, completed, created_at
FROM tasks
ORDER BY created_at DESC;

-- piko.name: CreateTask
-- piko.command: one
INSERT INTO tasks (title, completed, created_at)
VALUES (?, 0, ?)
RETURNING id, title, completed, created_at;

-- piko.name: ToggleComplete
-- piko.command: exec
UPDATE tasks SET completed = CASE WHEN completed = 0 THEN 1 ELSE 0 END
WHERE id = ?;

-- piko.name: DeleteTask
-- piko.command: exec
DELETE FROM tasks WHERE id = ?;
