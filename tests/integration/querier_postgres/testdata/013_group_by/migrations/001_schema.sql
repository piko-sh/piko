CREATE TABLE orders (
    id SERIAL PRIMARY KEY,
    customer TEXT NOT NULL,
    total INTEGER NOT NULL
);

CREATE TABLE order_items (
    id SERIAL PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders(id),
    product TEXT NOT NULL,
    quantity INTEGER NOT NULL
);
