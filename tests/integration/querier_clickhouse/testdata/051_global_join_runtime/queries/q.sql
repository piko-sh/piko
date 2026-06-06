-- piko.query(InsertUser, exec)
INSERT INTO users (id, name) VALUES ({id:UInt64}, {name:String});

-- piko.query(InsertSession, exec)
INSERT INTO sessions (id, user_id, duration) VALUES ({id:UInt64}, {user_id:UInt64}, {duration:UInt32});

-- piko.query(SessionsByUser, many)
SELECT u.id AS user_id, u.name AS user_name, s.duration AS duration
FROM users AS u
GLOBAL LEFT JOIN sessions AS s ON s.user_id = u.id
ORDER BY u.id, s.duration;
