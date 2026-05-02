-- piko.query(name: CreateAuthor, command: exec)
INSERT INTO authors (name, email, created_at) VALUES (?, ?, ?);

-- piko.query(name: ListAuthors, command: many)
SELECT id, name, email FROM authors ORDER BY name;

-- piko.query(name: GetAuthor, command: one)
SELECT id, name, email FROM authors WHERE id = ?;
