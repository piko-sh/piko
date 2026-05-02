-- piko.query(name: ExtractParts, command: one)
SELECT id, name,
    date_part('year', starts_at)::INTEGER AS start_year,
    date_part('month', starts_at)::INTEGER AS start_month,
    date_part('day', event_date)::INTEGER AS event_day
FROM events
WHERE id = $1;

-- piko.query(name: TruncateToMonth, command: many)
SELECT DATE_TRUNC('month', event_date)::DATE AS month, COUNT(*)::INTEGER AS event_count
FROM events
GROUP BY DATE_TRUNC('month', event_date)
ORDER BY month;

-- piko.query(name: EventDuration, command: many)
SELECT id, name, AGE(ends_at, starts_at)::TEXT AS duration
FROM events
ORDER BY id;
