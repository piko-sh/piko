CREATE TABLE posts (
    id INTEGER PRIMARY KEY,
    title TEXT NOT NULL,
    category TEXT NOT NULL,
    views INTEGER NOT NULL,
    published INTEGER NOT NULL DEFAULT 1
);
