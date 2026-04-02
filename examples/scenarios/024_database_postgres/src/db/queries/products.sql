-- piko.name: ListProducts
-- piko.command: many
SELECT id, name, category, price, created_at
FROM products
ORDER BY name;

-- piko.name: CreateProduct
-- piko.command: one
INSERT INTO products (name, category, price, created_at)
VALUES ($1, $2, $3, $4)
RETURNING id, name, category, price, created_at;

-- piko.name: GetProduct
-- piko.command: one
SELECT id, name, category, price, created_at
FROM products
WHERE id = $1;
