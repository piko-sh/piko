-- piko.query(name: GetLineItemTotal, command: one)
SELECT
  id,
  unit_price * quantity + COALESCE(unit_price * quantity * tax_rate, 0) - COALESCE(discount, 0) as total
FROM line_items WHERE id = $1
