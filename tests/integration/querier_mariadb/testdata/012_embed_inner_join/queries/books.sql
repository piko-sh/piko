-- piko.name: GetBookWithAuthor
-- piko.command: one
SELECT b.id, b.title, /* piko.embed(authors) */ a.id, a.name
FROM books b
INNER JOIN authors a ON a.id = b.author_id
WHERE b.id = ?;
