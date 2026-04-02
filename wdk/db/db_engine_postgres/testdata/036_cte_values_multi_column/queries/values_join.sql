-- piko.name: ValuesJoin
-- piko.command: many
WITH filter(target_id, target_status) AS (VALUES (1, 'active'), (2, 'pending'))
SELECT i.id, i.name, i.status
FROM items i
JOIN filter f ON f.target_id = i.id AND f.target_status = i.status;
