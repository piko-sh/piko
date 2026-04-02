-- piko.name: InsertCategory
-- piko.command: exec
INSERT INTO categories (id, name, parent_id) VALUES (?, ?, ?);

-- piko.name: GetSubtree
-- piko.command: many
WITH RECURSIVE subtree AS (
    SELECT id, name, parent_id, 0 AS depth FROM categories WHERE id = ?
    UNION ALL
    SELECT c.id, c.name, c.parent_id, s.depth + 1 FROM categories c JOIN subtree s ON c.parent_id = s.id
)
SELECT id, name, parent_id, depth FROM subtree ORDER BY depth, id;

-- piko.name: GetAncestors
-- piko.command: many
WITH RECURSIVE ancestors AS (
    SELECT id, name, parent_id, 0 AS depth FROM categories WHERE id = ?
    UNION ALL
    SELECT c.id, c.name, c.parent_id, a.depth + 1 FROM categories c JOIN ancestors a ON c.id = a.parent_id
)
SELECT id, name, depth FROM ancestors ORDER BY depth;

-- piko.name: ListRootCategories
-- piko.command: many
WITH roots AS (
    SELECT id, name FROM categories WHERE parent_id IS NULL
)
SELECT id, name FROM roots ORDER BY id;
