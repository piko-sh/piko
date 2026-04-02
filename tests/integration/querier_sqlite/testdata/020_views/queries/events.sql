-- piko.name: ListActiveEvents
-- piko.command: many
SELECT id, name, event_date FROM active_events ORDER BY event_date;

-- piko.name: GetActiveEvent
-- piko.command: one
SELECT id, name, event_date FROM active_events WHERE id = ?;
