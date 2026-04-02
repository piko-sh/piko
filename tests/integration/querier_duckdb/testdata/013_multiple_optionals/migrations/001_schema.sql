CREATE TABLE employees (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    department TEXT NOT NULL,
    level INTEGER NOT NULL,
    active INTEGER NOT NULL DEFAULT 1
);
