CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    price REAL NOT NULL
);

CREATE TABLE price_updates (
    product_id INTEGER NOT NULL,
    new_price REAL NOT NULL
);
