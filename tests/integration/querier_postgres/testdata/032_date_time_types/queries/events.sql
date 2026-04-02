-- piko.name: ExtractParts
-- piko.command: one
SELECT id, name,
    EXTRACT(YEAR FROM starts_at)::INTEGER AS start_year,
    EXTRACT(MONTH FROM starts_at)::INTEGER AS start_month,
    EXTRACT(DAY FROM event_date)::INTEGER AS event_day
FROM events
WHERE id = $1;

-- piko.name: TruncateToMonth
-- piko.command: many
SELECT DATE_TRUNC('month', event_date)::DATE AS month, COUNT(*)::INTEGER AS event_count
FROM events
GROUP BY DATE_TRUNC('month', event_date)
ORDER BY month;

-- piko.name: EventDuration
-- piko.command: many
SELECT id, name, AGE(ends_at, starts_at)::TEXT AS duration
FROM events
ORDER BY id;
