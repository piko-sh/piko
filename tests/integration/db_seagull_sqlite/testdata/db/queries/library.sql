-- piko.query(GetAuthor, one)
SELECT id, name, born_year FROM authors WHERE id = ?;

-- piko.query(ListAuthors, many)
SELECT id, name, born_year FROM authors ORDER BY id;

-- piko.query(CreateBook, one)
INSERT INTO books (author_id, title, price, rating)
VALUES (?, ?, ?, ?)
RETURNING id, title, price, rating;

-- piko.query(ListAuthorsWithBooks, many)
SELECT a.id AS author_id, a.name, b.id AS book_id, b.title
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
ORDER BY a.id, b.id;

-- piko.query(CountBooksByAuthor, one)
SELECT count(*) AS book_count FROM books WHERE author_id = ?;

-- piko.query(DeleteBook, exec)
DELETE FROM books WHERE id = ?;
