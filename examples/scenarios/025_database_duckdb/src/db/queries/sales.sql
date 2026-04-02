-- piko.name: SummaryStats
-- piko.command: one
SELECT
  COUNT(*)::INTEGER                         AS total_orders,
  SUM(amount * quantity)::DECIMAL(12, 2)    AS total_revenue,
  (AVG(amount * quantity))::DECIMAL(10, 2)  AS avg_order_value,
  COUNT(DISTINCT product)::INTEGER          AS unique_products
FROM sales;

-- piko.name: RevenueByCategory
-- piko.command: many
SELECT
  category,
  SUM(amount * quantity)::DECIMAL(12, 2)  AS revenue,
  SUM(quantity)::INTEGER                  AS units_sold
FROM sales
GROUP BY category
ORDER BY revenue DESC;

-- piko.name: RevenueByRegion
-- piko.command: many
SELECT
  region,
  SUM(amount * quantity)::DECIMAL(12, 2)  AS revenue,
  COUNT(*)::INTEGER                       AS order_count
FROM sales
GROUP BY region
ORDER BY revenue DESC;

-- piko.name: TopProducts
-- piko.command: many
SELECT
  product,
  category,
  SUM(quantity)::INTEGER                  AS total_units,
  SUM(amount * quantity)::DECIMAL(12, 2)  AS total_revenue
FROM sales
GROUP BY product, category
ORDER BY total_revenue DESC
LIMIT 5;

-- piko.name: MonthlySales
-- piko.command: many
SELECT
  strftime('%Y-%m', sold_at)  AS month,
  SUM(amount * quantity)::DECIMAL(12, 2)  AS revenue,
  SUM(quantity)::INTEGER                  AS units
FROM sales
GROUP BY month
ORDER BY month;
