CREATE TABLE customers (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer_id INTEGER NOT NULL REFERENCES customers(id),
    total INTEGER NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
);
