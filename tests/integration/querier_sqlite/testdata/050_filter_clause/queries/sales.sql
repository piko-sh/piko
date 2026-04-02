-- piko.name: GetCategorySummary
-- piko.command: many
SELECT
  category,
  COUNT(*) AS total_count,
  COUNT(*) FILTER (WHERE region = 'north') AS north_count,
  SUM(amount) FILTER (WHERE region = 'south') AS south_total
FROM sales
GROUP BY category
ORDER BY category;
