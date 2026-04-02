-- piko.name: FormattedSchedule
-- piko.command: many
SELECT
    id,
    title,
    DATE_FORMAT(scheduled_at, '%Y-%m-%d %H:%i') AS formatted_date,
    DATEDIFF(scheduled_at, CURDATE()) AS days_until
FROM appointments
WHERE scheduled_at >= CURDATE();
