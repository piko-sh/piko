-- piko.query(name: CheckExists, command: one)
-- ?1 as piko.param(target_id)
SELECT EXISTS(SELECT 1 FROM queue WHERE id = ?1) AS found;

-- piko.query(name: DequeueHighestPriority, command: one)
DELETE FROM queue WHERE id = (SELECT id FROM queue ORDER BY priority DESC LIMIT 1) RETURNING id, payload;
