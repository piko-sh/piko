-- piko.name: OrdersForceIndex
-- piko.command: many
SELECT id, customer_id, total, status
FROM orders FORCE INDEX (idx_orders_status)
WHERE status = ?;
