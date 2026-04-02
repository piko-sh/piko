CREATE TABLE authors (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE books (
    id SERIAL PRIMARY KEY,
    author_id INTEGER NOT NULL REFERENCES authors(id),
    title TEXT NOT NULL,
    published DATE
);
