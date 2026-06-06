-- piko.query(name: InsertOrder, command: exec)
INSERT INTO orders (customer, total) VALUES ($1, $2);

-- piko.query(name: ListOrders, command: many)
SELECT id, customer, total FROM orders ORDER BY id;
