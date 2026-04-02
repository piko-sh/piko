-- piko.name: ListByCategory
-- piko.command: many
SELECT id, name, price FROM products WHERE category = ? ORDER BY price ASC;

-- piko.name: GetBySku
-- piko.command: one
SELECT id, name, category, price FROM products WHERE sku = ?;

-- piko.name: ListByCategoryAndMaxPrice
-- piko.command: many
SELECT id, name, price FROM products WHERE category = ? AND price <= ? ORDER BY price ASC;
