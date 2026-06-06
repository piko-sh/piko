-- piko.query(name: SearchProducts, command: many)
-- ?1 as piko.param(category, optional: true)
-- ?2 as piko.param(page_size, default: 5, max: 20)
-- ?3 as piko.param(page_offset)
SELECT id, name, price, category FROM products WHERE (?1 IS NULL OR category = ?1) ORDER BY id ASC LIMIT ?2 OFFSET ?3;
