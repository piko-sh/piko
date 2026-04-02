-- piko.name: TreeWalk
-- piko.command: many
WITH RECURSIVE cat_tree AS (
    SELECT id, name, parent_id, 1 AS depth
    FROM categories
    WHERE parent_id IS NULL
    UNION ALL
    SELECT c.id, c.name, c.parent_id, ct.depth + 1
    FROM categories c
    JOIN cat_tree ct ON c.parent_id = ct.id
)
SELECT id, name, depth FROM cat_tree;
