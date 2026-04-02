-- piko.name: FindUser
-- piko.command: one
-- :name as piko.param
-- :email as piko.param
SELECT id, name, email, active FROM users WHERE name = :name AND email = :email;
