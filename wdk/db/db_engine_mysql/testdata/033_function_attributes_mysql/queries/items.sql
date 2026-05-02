-- piko.query(name: GetDoubled, command: one)
SELECT pure_double(quantity) AS doubled FROM items WHERE id = ?;

-- piko.query(name: GetItemCount, command: one)
SELECT get_item_count() AS total;
