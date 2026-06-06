-- piko.query(TopSpenders, many)
WITH user_totals AS (
    SELECT user_id, sum(total) AS lifetime_total FROM orders GROUP BY user_id
)
SELECT user_id, lifetime_total FROM user_totals ORDER BY lifetime_total DESC LIMIT 10;
