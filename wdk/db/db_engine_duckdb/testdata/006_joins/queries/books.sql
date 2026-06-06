-- piko.query(name: GetBooksWithAuthorsInner, command: many)
SELECT b.id, b.title, a.name AS author_name FROM books b INNER JOIN authors a ON b.author_id = a.id;

-- piko.query(name: GetBooksWithAuthorsLeft, command: many)
SELECT b.id, b.title, a.name AS author_name FROM books b LEFT JOIN authors a ON b.author_id = a.id;
