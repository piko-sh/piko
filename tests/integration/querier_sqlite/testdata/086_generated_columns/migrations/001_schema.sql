CREATE TABLE items (
    id INTEGER PRIMARY KEY,
    price INTEGER NOT NULL,
    quantity INTEGER NOT NULL,
    total INTEGER GENERATED ALWAYS AS (price * quantity) STORED,
    label TEXT GENERATED ALWAYS AS (printf('Item #%d', id)) STORED
);
