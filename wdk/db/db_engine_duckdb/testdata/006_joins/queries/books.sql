-- piko.name: GetBooksWithAuthorsInner
-- piko.command: many
SELECT b.id, b.title, a.name AS author_name FROM books b INNER JOIN authors a ON b.author_id = a.id;

-- piko.name: GetBooksWithAuthorsLeft
-- piko.command: many
SELECT b.id, b.title, a.name AS author_name FROM books b LEFT JOIN authors a ON b.author_id = a.id;
