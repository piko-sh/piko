-- piko.query(name: ListEvents, command: many)
SELECT id, name, payload, created_at FROM events WHERE name = ?
