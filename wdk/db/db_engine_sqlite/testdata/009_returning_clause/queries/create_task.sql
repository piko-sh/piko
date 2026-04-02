-- piko.name: CreateTask
-- piko.command: one
INSERT INTO tasks (title) VALUES (?) RETURNING id, title, done
