-- piko.query(name: GetOrCreateItem, command: one)
-- $1 as piko.param(id)
-- $2 as piko.param(name)
-- $3 as piko.param(priority)
WITH existing AS (
  SELECT id, name, priority FROM items WHERE id = $1
),
created AS (
  INSERT INTO items (id, name, priority)
  SELECT $1::uuid, $2::text, $3::int
  WHERE NOT EXISTS (SELECT 1 FROM existing)
  RETURNING id, name, priority
)
SELECT id, name, priority FROM existing
UNION ALL
SELECT id, name, priority FROM created;
