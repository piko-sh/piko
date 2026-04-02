-- piko.name: UpcomingAppointments
-- piko.command: many
SELECT id, title, scheduled_at, duration_minutes
FROM appointments
WHERE scheduled_at > NOW()
ORDER BY scheduled_at;
