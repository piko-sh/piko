-- piko.query(name: GetAllUsersUnion, command: many)
SELECT id, name FROM active_users UNION ALL SELECT id, name FROM archived_users;

-- piko.query(name: GetCommonUsers, command: many)
SELECT id, name FROM active_users INTERSECT SELECT id, name FROM archived_users;

-- piko.query(name: GetActiveOnlyUsers, command: many)
SELECT id, name FROM active_users EXCEPT SELECT id, name FROM archived_users;
