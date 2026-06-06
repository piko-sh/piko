-- piko.query(name: AllUsers, command: many)
SELECT id, name, email FROM active_users
UNION ALL
SELECT id, name, email FROM archived_users;
