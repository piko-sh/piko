-- piko.name: GetProductDetails
-- piko.command: one
SELECT
    id,
    name || ' (' || category || ')' AS display_name,
    price * (1.0 + tax_rate) AS price_with_tax,
    price * quantity AS total_value,
    quantity - 1 AS quantity_minus_one
FROM products
WHERE id = ?;
