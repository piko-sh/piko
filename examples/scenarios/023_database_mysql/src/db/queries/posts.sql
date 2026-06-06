-- piko.query(name: ListPublishedPosts, command: many)
SELECT p.id, p.title, p.body, p.created_at, a.name as author_name
FROM posts p
JOIN authors a ON p.author_id = a.id
WHERE p.published = TRUE
ORDER BY p.created_at DESC
LIMIT ?;

-- piko.query(name: GetPost, command: one)
SELECT p.id, p.title, p.body, p.published, p.created_at, a.name as author_name, a.email as author_email
FROM posts p
JOIN authors a ON p.author_id = a.id
WHERE p.id = ?;

-- piko.query(name: CreatePost, command: exec)
INSERT INTO posts (author_id, title, body, published, created_at)
VALUES (?, ?, ?, TRUE, ?);

-- piko.query(name: PublishPost, command: exec)
UPDATE posts SET published = TRUE WHERE id = ?;

-- piko.query(name: DeletePost, command: execrows)
DELETE FROM posts WHERE id = ?;

-- piko.query(name: GetPostCount, command: one)
SELECT COUNT(*) as total FROM posts WHERE published = TRUE;
