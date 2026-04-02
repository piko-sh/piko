-- piko.name: SearchProducts
-- piko.command: many
-- ?1 as piko.optional(category)
-- ?2 as piko.limit(page_size) default:5 max:20
-- ?3 as piko.offset(page_offset)
SELECT id, name, price, category FROM products WHERE (?1 IS NULL OR category = ?1) ORDER BY id ASC LIMIT ?2 OFFSET ?3;
