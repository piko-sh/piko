-- piko.name: InsertOrder
-- piko.command: exec
INSERT INTO orders (customer, total) VALUES ($1, $2);

-- piko.name: ListOrders
-- piko.command: many
SELECT id, customer, total FROM orders ORDER BY id;
