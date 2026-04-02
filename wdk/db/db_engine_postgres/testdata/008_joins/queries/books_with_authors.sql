-- piko.name: BooksWithAuthors
-- piko.command: many
SELECT b.id, b.title, a.name AS author_name
FROM books b
INNER JOIN authors a ON a.id = b.author_id
ORDER BY b.title;
