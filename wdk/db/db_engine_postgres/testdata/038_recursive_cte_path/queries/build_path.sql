-- piko.name: BuildPath
-- piko.command: many
WITH RECURSIVE path AS (
    SELECT id, name, name AS full_path
    FROM categories
    WHERE parent_id IS NULL
    UNION ALL
    SELECT c.id, c.name, p.full_path || ' > ' || c.name
    FROM categories c
    JOIN path p ON c.parent_id = p.id
)
SELECT id, full_path FROM path;
