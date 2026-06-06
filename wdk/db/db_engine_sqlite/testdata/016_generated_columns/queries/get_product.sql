-- piko.query(name: GetProduct, command: one)
SELECT id, name, price, quantity, total_value, display_name FROM products WHERE id = ?
