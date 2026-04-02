-- piko.name: GetEvent
-- piko.command: one
SELECT id, created_at, name, payload FROM events WHERE id = $1;
