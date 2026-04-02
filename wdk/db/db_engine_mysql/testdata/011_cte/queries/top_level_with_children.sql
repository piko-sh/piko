-- piko.name: TopLevelWithChildren
-- piko.command: many
WITH top_level AS (
    SELECT id, name FROM categories WHERE parent_id IS NULL
)
SELECT c.id, c.name, t.name AS parent_name
FROM categories c
INNER JOIN top_level t ON c.parent_id = t.id;
