-- piko.query(TreePath, many)
WITH RECURSIVE walk AS (
    SELECT id, parent_id FROM tree WHERE id = {root:UInt64}
    UNION ALL
    SELECT t.id, t.parent_id FROM tree t INNER JOIN walk w ON w.id = t.parent_id
)
SELECT id, parent_id FROM walk;
