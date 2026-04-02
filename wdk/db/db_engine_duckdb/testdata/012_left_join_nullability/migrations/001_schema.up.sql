CREATE TABLE orders (
    id INTEGER PRIMARY KEY,
    customer_id INTEGER NOT NULL
);

CREATE TABLE customers (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL
);
