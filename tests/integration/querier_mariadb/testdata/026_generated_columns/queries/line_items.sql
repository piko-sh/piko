-- piko.name: InsertLineItem
-- piko.command: exec
INSERT INTO line_items (product, quantity, unit_price, discount_pct) VALUES (?, ?, ?, ?);

-- piko.name: GetLineItem
-- piko.command: one
SELECT id, product, quantity, unit_price, total_price, discount_pct, discounted_price FROM line_items WHERE id = ?;

-- piko.name: ListLineItems
-- piko.command: many
SELECT id, product, quantity, unit_price, total_price, discount_pct, discounted_price FROM line_items ORDER BY id;
