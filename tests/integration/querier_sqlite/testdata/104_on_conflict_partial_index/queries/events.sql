-- piko.query(name: InsertEvent, command: one)
-- ?1 as piko.param(slug)
INSERT INTO events (slug) VALUES (?1)
ON CONFLICT (slug) WHERE archived = 0 DO NOTHING
RETURNING id, slug;
