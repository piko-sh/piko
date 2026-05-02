-- piko.query(name: CreateProduct, command: one)
INSERT INTO products (name, price, quantity) VALUES (?, ?, ?)
RETURNING id, name, price, quantity
