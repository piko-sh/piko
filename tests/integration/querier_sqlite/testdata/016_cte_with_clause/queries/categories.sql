-- piko.name: GetTopLevelWithChildCount
-- piko.command: many
WITH child_counts AS (
    SELECT parent_id, count(*) AS child_count
    FROM categories
    WHERE parent_id IS NOT NULL
    GROUP BY parent_id
)
SELECT c.id, c.name, COALESCE(cc.child_count, 0) AS child_count
FROM categories c
LEFT JOIN child_counts cc ON cc.parent_id = c.id
WHERE c.parent_id IS NULL
ORDER BY c.name;
