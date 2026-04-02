-- piko.name: CustomersWithOrders
-- piko.command: many
SELECT id, name
FROM customers c
WHERE EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.id);
