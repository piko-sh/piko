-- piko.name: GetUserByID
-- piko.command: one
-- @user_id as piko.param
SELECT id, name, email FROM users WHERE id = @user_id;

-- piko.name: InsertUser
-- piko.command: exec
-- @name as piko.param
-- @email as piko.param
INSERT INTO users (name, email) VALUES (@name, @email);
