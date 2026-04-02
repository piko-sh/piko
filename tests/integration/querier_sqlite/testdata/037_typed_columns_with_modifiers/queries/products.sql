-- piko.name: GetProduct
-- piko.command: one
-- ?1 as piko.param(product_id)
SELECT id, name, sku, price, active FROM products WHERE id = ?1;
