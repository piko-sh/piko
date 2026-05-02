-- piko.query(name: InsertLineItem, command: one)
INSERT INTO line_items (product, quantity, unit_price)
VALUES ($1, $2, $3)
RETURNING id, product, quantity, unit_price, total_price, display_name;

-- piko.query(name: ListLineItems, command: many)
SELECT id, product, quantity, unit_price, total_price, display_name
FROM line_items
ORDER BY id;
