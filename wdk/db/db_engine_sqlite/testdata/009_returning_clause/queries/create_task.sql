-- piko.query(name: CreateTask, command: one)
INSERT INTO tasks (title) VALUES (?) RETURNING id, title, done
