-- piko.name: GetEventsBetweenDates
-- piko.command: many
-- ?1 as piko.param(start_date)
-- ?2 as piko.param(end_date)
SELECT id, name, event_date FROM events WHERE event_date BETWEEN ?1 AND ?2 ORDER BY event_date;
