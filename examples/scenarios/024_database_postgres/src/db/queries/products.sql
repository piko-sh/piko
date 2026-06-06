-- piko.query(name: ListProducts, command: many)
SELECT id, name, category, price, created_at
FROM products
ORDER BY name;

-- piko.query(name: CreateProduct, command: one)
INSERT INTO products (name, category, price, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id, name, category, price, created_at;

-- piko.query(name: GetProduct, command: one)
SELECT id, name, category, price, created_at
FROM products
WHERE id = $1;
