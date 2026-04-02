-- piko.name: GetOrderStatus
-- piko.command: many
SELECT
  id,
  CASE status WHEN 'active' THEN amount WHEN 'discounted' THEN discount ELSE 0 END as effective_amount,
  CASE WHEN amount > 100 THEN 'large' WHEN amount > 50 THEN 'medium' END as size_label
FROM orders
