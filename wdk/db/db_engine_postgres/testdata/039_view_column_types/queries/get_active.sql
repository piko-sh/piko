-- piko.name: GetActiveUser
-- piko.command: one
SELECT id, name, email FROM active_users WHERE id = $1;
