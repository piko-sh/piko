-- piko.name: BooksWithAuthors
-- piko.command: many
SELECT b.id, b.title, a.name AS author_name
FROM books b
LEFT JOIN authors a ON b.author_id = a.id;
