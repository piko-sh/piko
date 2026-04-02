-- piko.name: ListPublishedPosts
-- piko.command: many
SELECT p.id, p.title, p.body, p.created_at, a.name as author_name
FROM posts p
JOIN authors a ON p.author_id = a.id
WHERE p.published = TRUE
ORDER BY p.created_at DESC
LIMIT ?;

-- piko.name: GetPost
-- piko.command: one
SELECT p.id, p.title, p.body, p.published, p.created_at, a.name as author_name, a.email as author_email
FROM posts p
JOIN authors a ON p.author_id = a.id
WHERE p.id = ?;

-- piko.name: CreatePost
-- piko.command: exec
INSERT INTO posts (author_id, title, body, published, created_at)
VALUES (?, ?, ?, TRUE, ?);

-- piko.name: PublishPost
-- piko.command: exec
UPDATE posts SET published = TRUE WHERE id = ?;

-- piko.name: DeletePost
-- piko.command: execrows
DELETE FROM posts WHERE id = ?;

-- piko.name: GetPostCount
-- piko.command: one
SELECT COUNT(*) as total FROM posts WHERE published = TRUE;
