-- piko.query(name: GetItem, command: one)
SELECT id, name, price, quantity FROM items WHERE id = ?;
