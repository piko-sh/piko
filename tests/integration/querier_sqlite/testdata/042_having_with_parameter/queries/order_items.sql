-- piko.name: GetBulkCategories
-- piko.command: many
-- ?1 as piko.param(minimum_quantity)
SELECT category, SUM(quantity) AS total_quantity
FROM order_items
GROUP BY category
HAVING SUM(quantity) >= ?1
ORDER BY total_quantity DESC;
