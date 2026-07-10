-- piko.query(name: InsertEventDoUpdate, command: one)
INSERT INTO events (slug, payload)
VALUES ($1, $2)
ON CONFLICT (slug) WHERE archived = false DO UPDATE SET payload = $3
RETURNING id, slug, payload;
