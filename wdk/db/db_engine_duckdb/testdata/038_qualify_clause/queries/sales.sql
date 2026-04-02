-- piko.name: GetLatestSalePerRegion
-- piko.command: many
SELECT id, region, amount, sale_date
FROM sales
QUALIFY row_number() OVER (PARTITION BY region ORDER BY sale_date DESC) = 1;
