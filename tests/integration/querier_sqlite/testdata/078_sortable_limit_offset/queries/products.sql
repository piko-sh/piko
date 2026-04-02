-- piko.name: ListProductsSorted
-- piko.command: many
-- ?1 as piko.sortable(order_by) columns:name,price
SELECT id, name, price, category FROM products

-- piko.name: ListProductsPaginated
-- piko.command: many
-- ?1 as piko.limit(page_size) default:5 max:20
-- ?2 as piko.offset(page_offset)
SELECT id, name, price, category FROM products ORDER BY id ASC LIMIT ?1 OFFSET ?2;
