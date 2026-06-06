-- piko.query(GetUser, one)
SELECT id, email FROM users WHERE id = ?;

-- piko.query(GetUserAsString, one)
SELECT HEX(id) AS id_hex, email FROM users WHERE email = ?;
