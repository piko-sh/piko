CREATE TYPE tag AS ENUM ('featured', 'sale', 'new');
CREATE TABLE products (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    tags tag[] NOT NULL
);
