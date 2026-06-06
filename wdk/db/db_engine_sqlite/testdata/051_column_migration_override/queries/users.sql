-- piko.query(GetUser, one)
SELECT id, email FROM users WHERE id = ?;

-- piko.query(GetUserAsString, one)
SELECT CAST(id AS TEXT) AS id_str, email FROM users WHERE email = ?;
