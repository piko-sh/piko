CREATE TABLE posts (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    category TEXT NOT NULL,
    views INTEGER NOT NULL,
    published INTEGER NOT NULL DEFAULT 1
);
