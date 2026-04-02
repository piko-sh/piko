-- piko.name: CreateProduct
-- piko.command: one
INSERT INTO products (name, price, quantity) VALUES (?, ?, ?)
RETURNING id, name, price, quantity
