-- piko.query(name: InsertReturning, command: one)
INSERT INTO items VALUES ($1, $2, $3) RETURNING id, name, price;

-- piko.query(name: UpdateReturning, command: one)
UPDATE items SET price = $2 WHERE id = $1 RETURNING id, name, price;

-- piko.query(name: DeleteReturning, command: one)
DELETE FROM items WHERE id = $1 RETURNING id, name;
