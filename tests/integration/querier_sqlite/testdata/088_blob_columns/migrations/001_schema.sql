CREATE TABLE files (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    content BLOB NOT NULL,
    size INTEGER NOT NULL
);
