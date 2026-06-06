-- piko.query(name: ListOrdersWithItems, command: many, group_by: orders.id)
-- piko.embed(orders, from: o)
-- piko.embed(order_items, from: i)
SELECT  o.id, o.customer, o.total,
        i.id, i.product, i.quantity
FROM orders o
LEFT JOIN order_items i ON i.order_id = o.id
ORDER BY o.id ASC, i.id ASC;
