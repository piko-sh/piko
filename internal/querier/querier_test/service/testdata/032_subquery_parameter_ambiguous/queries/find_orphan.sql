-- piko.name: FindOrphans
-- piko.command: many
SELECT id, total
FROM orders
WHERE id IN (SELECT id FROM customers WHERE email = $1);
