-- piko.name: GetProductSummary
-- piko.command: many
SELECT sub.product, sub.total_sales, sub.sale_count
FROM (
    SELECT product, SUM(amount) AS total_sales, COUNT(*) AS sale_count
    FROM sales
    GROUP BY product
) AS sub
ORDER BY sub.total_sales DESC;
