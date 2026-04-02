-- piko.name: CreateOrder
-- piko.command: one
INSERT INTO orders (product_id, quantity, total, ordered_at)
VALUES ($1, $2, $3, $4)
RETURNING id, product_id, quantity, total, ordered_at;

-- piko.name: ListRecentOrders
-- piko.command: many
SELECT o.id, o.quantity, o.total, o.ordered_at,
       p.name as product_name, p.category as product_category
FROM orders o
JOIN products p ON o.product_id = p.id
ORDER BY o.ordered_at DESC
LIMIT $1;
