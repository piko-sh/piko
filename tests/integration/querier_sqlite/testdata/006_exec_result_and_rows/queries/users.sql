-- piko.query(name: UpdateUserEmail, command: execrows)
UPDATE users SET email = ? WHERE id = ?;

-- piko.query(name: DeleteUser, command: execresult)
DELETE FROM users WHERE id = ?;
