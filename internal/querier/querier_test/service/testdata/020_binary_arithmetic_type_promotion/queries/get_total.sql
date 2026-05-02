-- piko.query(name: GetProductTotal, command: one)
SELECT
  price * quantity as total,
  price + discount as adjusted_price,
  quantity / 2 as half_quantity
FROM products WHERE id = $1
