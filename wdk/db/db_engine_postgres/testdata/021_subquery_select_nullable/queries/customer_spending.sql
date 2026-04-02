-- piko.name: CustomerSpending
-- piko.command: many
SELECT c.name, (SELECT SUM(o.amount) FROM orders o WHERE o.customer_id = c.id) AS total_spent
FROM customers c;
