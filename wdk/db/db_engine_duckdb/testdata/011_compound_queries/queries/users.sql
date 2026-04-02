-- piko.name: GetAllUsersUnion
-- piko.command: many
SELECT id, name FROM active_users UNION ALL SELECT id, name FROM archived_users;

-- piko.name: GetCommonUsers
-- piko.command: many
SELECT id, name FROM active_users INTERSECT SELECT id, name FROM archived_users;

-- piko.name: GetActiveOnlyUsers
-- piko.command: many
SELECT id, name FROM active_users EXCEPT SELECT id, name FROM archived_users;
