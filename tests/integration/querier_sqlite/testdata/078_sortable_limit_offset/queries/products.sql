-- piko.query(name: ListProductsSorted, command: many)
-- piko.sortable(order_by, columns: [name, price])
SELECT id, name, price, category FROM products

-- piko.query(name: ListProductsPaginated, command: many)
-- ?1 as piko.param(page_size, default: 5, max: 20)
-- ?2 as piko.param(page_offset)
SELECT id, name, price, category FROM products ORDER BY id ASC LIMIT ?1 OFFSET ?2;
