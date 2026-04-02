-- piko.name: GetWindowedSales
-- piko.command: many
SELECT
  id,
  region,
  amount,
  SUM(amount) OVER (PARTITION BY region) as region_total,
  ROW_NUMBER() OVER (ORDER BY amount DESC) as rank
FROM sales
