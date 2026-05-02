-- piko.query(GetUserWithAccount, one)
SELECT u.id, u.email, a.balance
FROM users u INNER JOIN accounts a ON a.user_id = u.id
WHERE u.id = {uid:UInt64};
