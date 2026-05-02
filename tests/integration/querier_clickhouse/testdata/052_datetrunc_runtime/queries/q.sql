-- piko.query(InsertEvent, exec)
INSERT INTO events (id, ts, amount) VALUES ({id:UInt64}, {ts:DateTime}, {amount:UInt32});

-- piko.query(MonthlyTotals, many)
SELECT
    dateTrunc('month', ts) AS month_bucket,
    sum(amount)            AS total
FROM events
GROUP BY month_bucket
ORDER BY month_bucket;
