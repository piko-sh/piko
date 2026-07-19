-- piko.query(name: InsertEventDoNothing, command: one, optional: true)
INSERT INTO events (slug, payload)
VALUES ($1, $2)
ON CONFLICT (slug) WHERE archived = false DO NOTHING
RETURNING id, slug;
