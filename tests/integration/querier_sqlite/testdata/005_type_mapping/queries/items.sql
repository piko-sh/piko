-- piko.query(name: GetItem, command: one)
SELECT id, name, price, quantity, active FROM items WHERE id = ?;

-- piko.query(name: ListItems, command: many)
SELECT id, name, price, quantity, active FROM items ORDER BY id;
