-- piko.name: GetCustomerStats
-- piko.command: many
WITH order_totals AS (
    SELECT customer_id, SUM(amount) AS total_amount, COUNT(*) AS order_count
    FROM orders
    GROUP BY customer_id
),
ranked AS (
    SELECT customer_id, total_amount, order_count,
           CASE WHEN total_amount > 100 THEN 'high' ELSE 'low' END AS tier
    FROM order_totals
)
SELECT customer_id, total_amount, order_count, tier FROM ranked ORDER BY customer_id;
