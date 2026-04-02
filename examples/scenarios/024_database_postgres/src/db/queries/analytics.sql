-- piko.name: GetCategorySummary
-- piko.command: many
WITH category_totals AS (
  SELECT
    p.category,
    SUM(o.total) as revenue,
    COUNT(*) as order_count
  FROM orders o
  JOIN products p ON o.product_id = p.id
  GROUP BY p.category
)
SELECT
  category,
  revenue,
  order_count,
  SUM(revenue) OVER (ORDER BY revenue DESC) as running_total
FROM category_totals
ORDER BY revenue DESC;

-- piko.name: GetTopProducts
-- piko.command: many
SELECT
  p.name,
  p.category,
  SUM(o.total) as revenue,
  COUNT(*) as order_count,
  ROW_NUMBER() OVER (PARTITION BY p.category ORDER BY SUM(o.total) DESC) as category_rank
FROM products p
JOIN orders o ON p.id = o.product_id
GROUP BY p.id, p.name, p.category
ORDER BY revenue DESC
LIMIT $1;

-- piko.name: GetDailyRevenue
-- piko.command: many
SELECT
  (o.ordered_at / 86400) * 86400 as day_timestamp,
  SUM(o.total) as revenue,
  COUNT(*) as order_count
FROM orders o
GROUP BY (o.ordered_at / 86400)
ORDER BY day_timestamp DESC
LIMIT $1;
