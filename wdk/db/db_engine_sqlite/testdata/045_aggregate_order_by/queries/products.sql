-- piko.name: GetCategoryProducts
-- piko.command: many
SELECT
  category,
  GROUP_CONCAT(name ORDER BY name ASC) AS sorted_names,
  GROUP_CONCAT(name, ',' ORDER BY price DESC) AS by_price
FROM products
GROUP BY category;
