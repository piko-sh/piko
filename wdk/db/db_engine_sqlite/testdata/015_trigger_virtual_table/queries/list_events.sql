-- piko.name: ListEvents
-- piko.command: many
SELECT id, name, payload, created_at FROM events WHERE name = ?
