-- piko.name: GetCategoryTree
-- piko.command: many
WITH RECURSIVE category_tree AS (
    SELECT id, name, parent_id, 0 AS depth
    FROM categories
    WHERE parent_id IS NULL
    UNION ALL
    SELECT c.id, c.name, c.parent_id, ct.depth + 1
    FROM categories c
    INNER JOIN category_tree ct ON ct.id = c.parent_id
)
SELECT id, name, parent_id, depth
FROM category_tree
ORDER BY depth, id;

-- piko.name: GetSubtree
-- piko.command: many
WITH RECURSIVE subtree AS (
    SELECT id, name, parent_id, 0 AS depth
    FROM categories
    WHERE id = $1
    UNION ALL
    SELECT c.id, c.name, c.parent_id, s.depth + 1
    FROM categories c
    INNER JOIN subtree s ON s.id = c.parent_id
)
SELECT id, name, parent_id, depth
FROM subtree
ORDER BY depth, id;

-- piko.name: GetLeafCategories
-- piko.command: many
WITH children AS (
    SELECT DISTINCT parent_id FROM categories WHERE parent_id IS NOT NULL
)
SELECT c.id, c.name
FROM categories c
LEFT JOIN children ch ON ch.parent_id = c.id
WHERE ch.parent_id IS NULL
ORDER BY c.id;
