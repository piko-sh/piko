-- piko.query(name: UserStats, command: one)
SELECT COUNT(*) AS total_users, AVG(age) AS average_age, MAX(age) AS max_age FROM users;
