-- piko.name: SalesSummary
-- piko.command: one
SELECT
    COUNT(*) AS total_sales,
    SUM(quantity) AS total_quantity,
    AVG(unit_price) AS average_price,
    MIN(unit_price) AS lowest_price,
    MAX(unit_price) AS highest_price
FROM sales;
