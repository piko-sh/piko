-- piko.name: ListProductsPaginated
-- piko.command: many
-- ?1 as piko.limit(page_size)
-- ?2 as piko.offset(page_offset)
SELECT id, name, price FROM products ORDER BY id LIMIT ?1 OFFSET ?2;

-- piko.name: ListProductsByMinPrice
-- piko.command: many
-- ?1 as piko.param(minimum_price)
-- ?2 as piko.limit(page_size)
-- ?3 as piko.offset(page_offset)
SELECT id, name, price FROM products WHERE price >= ? ORDER BY price LIMIT ? OFFSET ?;
