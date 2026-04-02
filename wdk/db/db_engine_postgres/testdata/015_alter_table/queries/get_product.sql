-- piko.name: GetProduct
-- piko.command: one
SELECT id, name, price, description, active FROM products WHERE id = $1;
