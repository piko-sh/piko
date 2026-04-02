-- piko.name: InsertEvent
-- piko.command: exec
INSERT INTO events (title, event_date, start_time, created_at) VALUES (?, ?, ?, ?);

-- piko.name: GetEvent
-- piko.command: one
SELECT id, title, event_date, start_time, created_at FROM events WHERE id = ?;

-- piko.name: GetFormattedDate
-- piko.command: one
SELECT id, title, DATE_FORMAT(event_date, '%W, %M %d, %Y') AS formatted_date FROM events WHERE id = ?;

-- piko.name: GetDaysBetween
-- piko.command: one
SELECT DATEDIFF(e2.event_date, e1.event_date) AS days_apart
FROM events e1, events e2
WHERE e1.id = ? AND e2.id = ?;

-- piko.name: GetHoursBetween
-- piko.command: one
SELECT TIMESTAMPDIFF(HOUR, e1.created_at, e2.created_at) AS hours_apart
FROM events e1, events e2
WHERE e1.id = ? AND e2.id = ?;

-- piko.name: ListByDate
-- piko.command: many
SELECT id, title, event_date FROM events ORDER BY event_date;
