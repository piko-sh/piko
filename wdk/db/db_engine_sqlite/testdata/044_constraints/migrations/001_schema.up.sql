CREATE TABLE categories (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    category_id INTEGER NOT NULL,
    price REAL NOT NULL CHECK (price > 0),
    FOREIGN KEY (category_id) REFERENCES categories(id)
);
