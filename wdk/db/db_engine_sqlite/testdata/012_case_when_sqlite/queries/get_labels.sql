-- piko.name: GetOrderLabels
-- piko.command: many
SELECT
  id,
  CASE WHEN amount > 100 THEN 'large' WHEN amount > 50 THEN 'medium' ELSE 'small' END as size_with_else,
  CASE WHEN status = 'active' THEN amount END as active_amount
FROM orders
