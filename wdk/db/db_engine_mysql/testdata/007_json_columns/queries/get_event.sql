-- piko.name: GetEvent
-- piko.command: one
SELECT id, event_type, payload, created_at FROM events WHERE id = ?;
