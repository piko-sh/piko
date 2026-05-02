-- piko.query(name: GetUserByEmail, command: one)
-- :email as piko.param
SELECT id, name, email FROM users WHERE email = :email;

-- piko.query(name: InsertUser, command: exec)
-- :user_name as piko.param
-- :user_email as piko.param
INSERT INTO users (name, email) VALUES (:user_name, :user_email);
