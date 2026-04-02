-- piko.name: GetHighValueOrders
-- piko.command: many
WITH high_value AS (
  SELECT id, customer_name, amount
  FROM orders
  WHERE amount > ?
)
SELECT id, customer_name, amount FROM high_value
