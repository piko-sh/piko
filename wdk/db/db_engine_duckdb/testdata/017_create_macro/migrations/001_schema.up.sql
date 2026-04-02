CREATE TABLE products (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    price DOUBLE NOT NULL
);

CREATE MACRO discount(amount, pct) AS amount * (1.0 - pct / 100.0);
