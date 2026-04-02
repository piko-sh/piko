-- piko.name: GetSubtree
-- piko.command: many
WITH RECURSIVE subtree AS (
    SELECT id, name, parent_id, 0 AS depth FROM categories WHERE id = $1
    UNION ALL
    SELECT c.id, c.name, c.parent_id, s.depth + 1 FROM categories c INNER JOIN subtree s ON c.parent_id = s.id
)
SELECT id, name, parent_id, depth FROM subtree ORDER BY depth, id;

-- piko.name: GetAncestors
-- piko.command: many
WITH RECURSIVE ancestors AS (
    SELECT id, name, parent_id FROM categories WHERE id = $1
    UNION ALL
    SELECT c.id, c.name, c.parent_id FROM categories c INNER JOIN ancestors a ON c.id = a.parent_id
)
SELECT id, name, parent_id FROM ancestors ORDER BY id;
