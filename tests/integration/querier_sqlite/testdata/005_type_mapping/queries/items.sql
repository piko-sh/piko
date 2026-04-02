-- piko.name: GetItem
-- piko.command: one
SELECT id, name, price, quantity, active FROM items WHERE id = ?;

-- piko.name: ListItems
-- piko.command: many
SELECT id, name, price, quantity, active FROM items ORDER BY id;
