-- piko.name: GetUserByNameAndEmail
-- piko.command: one
SELECT id, name, email FROM users WHERE name = :name AND email = :email;
