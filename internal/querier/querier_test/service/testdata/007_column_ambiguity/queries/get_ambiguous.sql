-- piko.query(name: GetAmbiguous, command: one)
SELECT u.id, name FROM users u JOIN posts p ON p.author_id = u.id
