CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    sku TEXT NOT NULL,
    name TEXT NOT NULL,
    category TEXT NOT NULL,
    price INTEGER NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    attributes JSONB NOT NULL DEFAULT '{}'
);
