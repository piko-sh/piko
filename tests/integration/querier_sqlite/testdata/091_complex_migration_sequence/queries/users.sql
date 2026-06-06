-- piko.query(name: InsertUser, command: exec)
INSERT INTO users (id, name, email, role) VALUES (?, ?, ?, ?);

-- piko.query(name: InsertPost, command: exec)
INSERT INTO posts (id, user_id, title, body) VALUES (?, ?, ?, ?);

-- piko.query(name: GetUserPostCounts, command: many)
SELECT id, name, role, post_count FROM user_post_counts ORDER BY id;

-- piko.query(name: ListUsersByRole, command: many)
SELECT id, name, email FROM users WHERE role = ? ORDER BY id;
