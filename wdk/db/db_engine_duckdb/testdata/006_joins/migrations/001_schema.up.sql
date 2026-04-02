CREATE TABLE authors (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL
);

CREATE TABLE books (
    id INTEGER PRIMARY KEY,
    title VARCHAR NOT NULL,
    author_id INTEGER REFERENCES authors(id)
);
