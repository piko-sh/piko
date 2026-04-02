-- piko.name: SelectUserColumns
-- piko.command: many
SELECT u.* FROM users u JOIN posts p ON p.author_id = u.id
