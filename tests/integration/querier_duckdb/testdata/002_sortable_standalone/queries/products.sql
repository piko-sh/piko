-- piko.name: ListProductsSorted
-- piko.command: many
-- $1 as piko.sortable(order_by) columns:name,price,category
SELECT id, name, price, category FROM products
