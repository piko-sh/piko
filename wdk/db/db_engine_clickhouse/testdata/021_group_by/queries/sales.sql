-- piko.query(SalesByRegion, many)
SELECT region, sum(amount) AS total
FROM sales
GROUP BY region
ORDER BY total DESC;
