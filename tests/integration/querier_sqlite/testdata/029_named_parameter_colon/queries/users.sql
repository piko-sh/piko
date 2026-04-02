-- piko.name: GetUserByEmail
-- piko.command: one
-- :email as piko.param
SELECT id, name, email FROM users WHERE email = :email;

-- piko.name: InsertUser
-- piko.command: exec
-- :user_name as piko.param
-- :user_email as piko.param
INSERT INTO users (name, email) VALUES (:user_name, :user_email);
