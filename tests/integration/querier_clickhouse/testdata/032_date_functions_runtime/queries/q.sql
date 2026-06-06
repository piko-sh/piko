-- piko.query(WithYear, many)
SELECT id, toYear(ts) AS year, formatDateTime(ts, '%Y-%m-%d') AS day FROM events ORDER BY id;
