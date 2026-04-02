-- piko.name: GetProductBySku
-- piko.command: one
SELECT id, sku, name, category, price
FROM products
WHERE sku = $1;

-- piko.name: ListActiveByCategory
-- piko.command: many
SELECT id, sku, name, price
FROM products
WHERE category = $1 AND active = true
ORDER BY price;

-- piko.name: FindByAttributes
-- piko.command: many
SELECT id, sku, name, attributes
FROM products
WHERE attributes @> $1::jsonb
ORDER BY id;
