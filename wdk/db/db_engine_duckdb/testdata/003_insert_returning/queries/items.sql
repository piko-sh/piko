-- piko.query(name: InsertItemReturningID, command: one)
INSERT INTO items (id, name, price) VALUES ($1, $2, $3) RETURNING id;

-- piko.query(name: InsertItemReturningAll, command: one)
INSERT INTO items (id, name, price) VALUES ($1, $2, $3) RETURNING *;
