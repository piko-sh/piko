-- piko.name: ListOrdersWithCategory
-- piko.command: many
SELECT
    id,
    status,
    total,
    CASE
        WHEN total >= 100.0 THEN 'high'
        WHEN total >= 50.0 THEN 'medium'
        ELSE 'low'
    END AS price_category,
    CASE
        WHEN status = 'shipped' THEN total
    END AS shipped_total
FROM orders
ORDER BY id;
