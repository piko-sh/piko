-- piko.name: GetProduct
-- piko.command: one
SELECT id, name, price, quantity FROM products WHERE id = ?;
