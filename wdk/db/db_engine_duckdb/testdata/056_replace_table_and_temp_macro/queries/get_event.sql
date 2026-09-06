-- piko.query(name: GetEvent, command: one)
SELECT id, slug, payload, bump(id) AS next_id
FROM events
WHERE id = $1;
