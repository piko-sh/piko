-- piko.name: CustomersWithoutOrders
-- piko.command: many
SELECT id, name
FROM customers c
WHERE NOT EXISTS (SELECT 1 FROM orders o WHERE o.customer_id = c.id);
