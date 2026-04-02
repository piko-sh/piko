-- piko.name: GetCategoryStats
-- piko.command: many
SELECT
  category,
  count(*) AS item_count,
  lower(category) AS category_lower,
  avg(price) AS average_price
FROM items
GROUP BY category
