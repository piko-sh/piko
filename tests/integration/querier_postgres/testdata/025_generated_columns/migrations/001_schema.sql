CREATE TABLE line_items (
    id SERIAL PRIMARY KEY,
    product TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    unit_price INTEGER NOT NULL,
    total_price INTEGER GENERATED ALWAYS AS (quantity * unit_price) STORED,
    display_name TEXT GENERATED ALWAYS AS (product || ' x' || quantity) STORED
);
