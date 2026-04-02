-- piko.name: CreateAuthor
-- piko.command: exec
INSERT INTO authors (name, email, created_at) VALUES (?, ?, ?);

-- piko.name: ListAuthors
-- piko.command: many
SELECT id, name, email FROM authors ORDER BY name;

-- piko.name: GetAuthor
-- piko.command: one
SELECT id, name, email FROM authors WHERE id = ?;
