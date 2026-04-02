-- piko.name: CategoryTree
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
SELECT id, name, depth FROM category_tree ORDER BY depth, name;
