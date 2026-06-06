-- piko.query(InsertEvent, exec)
INSERT INTO events (id, user_id, amount) VALUES ({id:UInt64}, {user_id:UInt64}, {amount:UInt32});

-- piko.query(UserTotals, many)
SELECT user_id, sum(amount) AS total
FROM events
GROUP BY user_id
ORDER BY user_id;
