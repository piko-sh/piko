-- piko.name: InsertItemReturningID
-- piko.command: one
INSERT INTO items (id, name, price) VALUES ($1, $2, $3) RETURNING id;

-- piko.name: InsertItemReturningAll
-- piko.command: one
INSERT INTO items (id, name, price) VALUES ($1, $2, $3) RETURNING *;
