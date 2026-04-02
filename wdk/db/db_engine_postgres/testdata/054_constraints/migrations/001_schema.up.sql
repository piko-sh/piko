CREATE TABLE categories (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);

CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    category_id INTEGER NOT NULL,
    price NUMERIC(10, 2) NOT NULL CHECK (price > 0),
    CONSTRAINT fk_category FOREIGN KEY (category_id) REFERENCES categories(id)
);
