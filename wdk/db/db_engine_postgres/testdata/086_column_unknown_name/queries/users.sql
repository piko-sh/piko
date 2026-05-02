-- piko.query(GetUser, one)
-- piko.column(does_not_exist, type: text)
SELECT id, email FROM users WHERE id = $1;
