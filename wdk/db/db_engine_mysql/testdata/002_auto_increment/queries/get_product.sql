-- piko.query(name: GetProduct, command: one)
SELECT id, name, price, quantity FROM products WHERE id = ?;
