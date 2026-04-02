-- piko.name: GetDoubled
-- piko.command: one
SELECT pure_double(quantity) AS doubled FROM items WHERE id = ?;

-- piko.name: GetItemCount
-- piko.command: one
SELECT get_item_count() AS total;
