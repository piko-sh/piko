-- piko.query(name: SelectAllColumns, command: many)
SELECT u.*, p.* FROM users u JOIN posts p ON p.author_id = u.id
