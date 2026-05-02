-- piko.query(name: RightJoinQuery, command: many)
SELECT u.name, p.title FROM users u RIGHT JOIN posts p ON p.author_id = u.id
