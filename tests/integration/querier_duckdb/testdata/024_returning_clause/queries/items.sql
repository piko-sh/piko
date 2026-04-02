-- piko.name: InsertReturning
-- piko.command: one
INSERT INTO items VALUES ($1, $2, $3) RETURNING id, name, price;

-- piko.name: UpdateReturning
-- piko.command: one
UPDATE items SET price = $2 WHERE id = $1 RETURNING id, name, price;

-- piko.name: DeleteReturning
-- piko.command: one
DELETE FROM items WHERE id = $1 RETURNING id, name;
