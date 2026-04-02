-- piko.name: GetFilteredItems
-- piko.command: many
SELECT
  id,
  name,
  category IS NULL as is_uncategorised,
  name IS NOT NULL as has_name,
  category IN ('electronics', 'books', 'toys') as is_popular_category,
  price BETWEEN 10 AND 100 as is_mid_range
FROM items
