-- piko.name: ListItems
-- piko.command: many
-- ?1 as piko.limit(page_size) default:3 max:10
-- ?2 as piko.offset(page_offset)
SELECT id, name FROM items ORDER BY id ASC LIMIT ?1 OFFSET ?2;
