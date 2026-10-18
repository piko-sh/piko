CREATE TABLE authors (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL,
    born_year  INTEGER
);

CREATE TABLE books (
    id         INTEGER PRIMARY KEY,
    author_id  INTEGER NOT NULL REFERENCES authors (id),
    title      TEXT    NOT NULL,
    price      NUMERIC NOT NULL,
    metadata   TEXT
);
