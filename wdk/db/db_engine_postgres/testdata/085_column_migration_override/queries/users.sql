-- piko.query(GetUser, one)
SELECT id, email FROM users WHERE id = $1;

-- piko.query(GetUserAsString, one)
SELECT id::text AS id_str, email FROM users WHERE email = $1;
