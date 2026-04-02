-- piko.name: UpdateItemPrice
-- piko.command: one
UPDATE items SET price = $2 WHERE id = $1 RETURNING id, name, price;
