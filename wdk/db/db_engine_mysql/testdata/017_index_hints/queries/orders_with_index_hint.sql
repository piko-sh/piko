-- piko.name: OrdersWithIndexHint
-- piko.command: many
SELECT id, customer_id, total, status
FROM orders USE INDEX (idx_orders_customer)
WHERE customer_id = ?;
