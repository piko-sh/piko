CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    amount NUMERIC(10,2) NOT NULL
);
