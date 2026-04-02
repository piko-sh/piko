-- piko.name: UpdateUserEmail
-- piko.command: execrows
UPDATE users SET email = ? WHERE id = ?;

-- piko.name: DeleteUser
-- piko.command: execresult
DELETE FROM users WHERE id = ?;
