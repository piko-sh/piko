-- piko.name: GetProductTotals
-- piko.command: many
SELECT
  id,
  price * quantity as subtotal,
  price * COALESCE(tax_rate, 0.0) as tax_amount
FROM products
