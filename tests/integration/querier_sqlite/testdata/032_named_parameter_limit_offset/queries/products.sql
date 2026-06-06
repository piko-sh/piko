-- piko.query(name: ListProductsPaginated, command: many)
-- :page_size as piko.param
-- :page_offset as piko.param
SELECT id, name, price FROM products ORDER BY id LIMIT :page_size OFFSET :page_offset;
