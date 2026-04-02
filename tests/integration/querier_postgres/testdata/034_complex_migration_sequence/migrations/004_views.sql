CREATE VIEW customer_order_summary AS
SELECT c.id AS customer_id, c.name, c.email,
    COUNT(o.id)::INTEGER AS order_count,
    COALESCE(SUM(o.total), 0)::INTEGER AS total_spent
FROM customers c
LEFT JOIN orders o ON o.customer_id = c.id
GROUP BY c.id, c.name, c.email;
