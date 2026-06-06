-- piko.query(name: ListByCategory, command: many)
SELECT id, name, price FROM products WHERE category = ? ORDER BY price ASC;

-- piko.query(name: GetBySku, command: one)
SELECT id, name, category, price FROM products WHERE sku = ?;

-- piko.query(name: ListByCategoryAndMaxPrice, command: many)
SELECT id, name, price FROM products WHERE category = ? AND price <= ? ORDER BY price ASC;
