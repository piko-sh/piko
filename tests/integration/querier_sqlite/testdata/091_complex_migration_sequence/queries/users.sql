-- piko.name: InsertUser
-- piko.command: exec
INSERT INTO users (id, name, email, role) VALUES (?, ?, ?, ?);

-- piko.name: InsertPost
-- piko.command: exec
INSERT INTO posts (id, user_id, title, body) VALUES (?, ?, ?, ?);

-- piko.name: GetUserPostCounts
-- piko.command: many
SELECT id, name, role, post_count FROM user_post_counts ORDER BY id;

-- piko.name: ListUsersByRole
-- piko.command: many
SELECT id, name, email FROM users WHERE role = ? ORDER BY id;
