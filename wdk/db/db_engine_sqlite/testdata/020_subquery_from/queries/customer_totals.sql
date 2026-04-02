-- piko.name: GetCustomerTotals
-- piko.command: many
SELECT sub.customer_id, sub.total_amount
FROM (SELECT customer_id, sum(amount) AS total_amount FROM orders WHERE status = ? GROUP BY customer_id) sub
WHERE sub.total_amount > ?
