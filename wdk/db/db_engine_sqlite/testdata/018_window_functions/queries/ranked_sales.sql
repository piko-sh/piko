-- piko.name: GetRankedSales
-- piko.command: many
SELECT
  id,
  region,
  product,
  amount,
  row_number() OVER (PARTITION BY region ORDER BY amount DESC) AS rank_in_region,
  lag(amount, 1) OVER (PARTITION BY region ORDER BY sale_date) AS previous_amount,
  sum(amount) OVER (ORDER BY sale_date ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW) AS running_total
FROM sales
WHERE region = ?
