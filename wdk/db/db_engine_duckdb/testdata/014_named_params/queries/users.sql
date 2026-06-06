-- piko.query(name: GetUserByNameAndEmail, command: one)
SELECT id, name, email FROM users WHERE name = :name AND email = :email;
