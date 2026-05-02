-- piko.query(name: ListItems, command: many)
-- $1 as piko.param(page_size, default: 3, max: 10)
-- $2 as piko.param(page_offset)
SELECT id, name FROM items ORDER BY id ASC LIMIT $1 OFFSET $2;
