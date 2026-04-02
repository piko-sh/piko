-- piko.name: ListOrdersWithItems
-- piko.command: many
-- piko.group_by: orders.id
SELECT /* piko.embed(orders) */ o.id, o.customer, o.total,
       /* piko.embed(order_items) */ i.id, i.product, i.quantity
FROM orders o
LEFT JOIN order_items i ON i.order_id = o.id
ORDER BY o.id ASC, i.id ASC;
