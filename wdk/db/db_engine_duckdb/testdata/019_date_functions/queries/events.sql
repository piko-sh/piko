-- piko.name: GetEventsWithDateParts
-- piko.command: many
SELECT id, name, date_part('year', occurred_at) AS event_year FROM events;
