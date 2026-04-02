-- piko.name: EventsInRange
-- piko.command: many
SELECT id, name, event_date FROM events
WHERE event_date >= $1::date AND event_date <= $2::date
ORDER BY event_date;
