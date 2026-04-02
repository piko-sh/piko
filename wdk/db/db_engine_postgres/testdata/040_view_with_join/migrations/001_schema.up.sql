CREATE TABLE authors (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    author_id INTEGER NOT NULL REFERENCES authors(id)
);
CREATE VIEW book_details AS
    SELECT b.id, b.title, a.name AS author_name
    FROM books b
    JOIN authors a ON a.id = b.author_id;
