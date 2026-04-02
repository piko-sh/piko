-- piko.name: FullJoinQuery
-- piko.command: many
SELECT u.name, p.title FROM users u FULL OUTER JOIN posts p ON p.author_id = u.id
