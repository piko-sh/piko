-- piko.query(name: GetBookWithAuthor, command: one)
-- piko.embed(authors, from: a)
SELECT b.id, b.title,  a.id, a.name
FROM books b
INNER JOIN authors a ON a.id = b.author_id
WHERE b.id = ?1;
