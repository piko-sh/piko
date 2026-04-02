-- piko.name: InsertProduct
-- piko.command: exec
INSERT INTO products (name, data) VALUES (?, ?);

-- piko.name: GetProductPrice
-- piko.command: one
SELECT id, name, JSON_EXTRACT(data, '$.price') AS price FROM products WHERE id = ?;

-- piko.name: GetProductCategory
-- piko.command: one
SELECT id, name, data->>'$.category' AS category FROM products WHERE id = ?;

-- piko.name: FindByCategory
-- piko.command: many
SELECT id, name, data->>'$.category' AS category FROM products WHERE JSON_CONTAINS(data, JSON_OBJECT('category', ?)) ORDER BY id;

-- piko.name: BuildSummary
-- piko.command: one
SELECT id, JSON_OBJECT('product_name', name, 'product_price', JSON_EXTRACT(data, '$.price')) AS summary FROM products WHERE id = ?;
