-- piko.name: CategoryStats
-- piko.command: many
SELECT s.category, s.product_count, s.avg_price
FROM (
    SELECT category, COUNT(*) AS product_count, AVG(price) AS avg_price
    FROM products
    GROUP BY category
) AS s
ORDER BY s.avg_price DESC;
