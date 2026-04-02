-- piko.name: ListProductsPaginated
-- piko.command: many
-- :page_size as piko.limit
-- :page_offset as piko.offset
SELECT id, name, price FROM products ORDER BY id LIMIT :page_size OFFSET :page_offset;
