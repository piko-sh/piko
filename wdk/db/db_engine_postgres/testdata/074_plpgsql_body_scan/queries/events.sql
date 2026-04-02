-- piko.name: GetEventCount
-- piko.command: one
SELECT count_events() AS total;

-- piko.name: ListEvents
-- piko.command: many
SELECT id, name, occurred_at FROM events;
