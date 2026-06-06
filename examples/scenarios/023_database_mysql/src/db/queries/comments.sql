-- piko.query(name: GetCommentsForPost, command: many)
SELECT id, author_name, body, created_at
FROM comments
WHERE post_id = ?
ORDER BY created_at ASC;

-- piko.query(name: CreateComment, command: exec)
INSERT INTO comments (post_id, author_name, body, created_at)
VALUES (?, ?, ?, ?);

-- piko.query(name: GetCommentCount, command: one)
SELECT COUNT(*) as total FROM comments WHERE post_id = ?;
