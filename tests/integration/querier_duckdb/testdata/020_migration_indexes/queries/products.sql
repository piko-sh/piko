-- piko.name: ListByCategory
-- piko.command: many
SELECT id, name, category, price, sku FROM products WHERE category = $1 ORDER BY id;

-- piko.name: GetBySku
-- piko.command: one
SELECT id, name, category, price, sku FROM products WHERE sku = $1;
