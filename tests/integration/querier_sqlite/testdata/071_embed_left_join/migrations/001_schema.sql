CREATE TABLE authors (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE books (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    author_id INTEGER NOT NULL REFERENCES authors(id)
);

CREATE TABLE reviews (
    id INTEGER PRIMARY KEY,
    book_id INTEGER NOT NULL REFERENCES books(id),
    rating INTEGER NOT NULL
);
