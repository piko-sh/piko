-- piko.name: GetCommentsForPost
-- piko.command: many
SELECT id, author_name, body, created_at
FROM comments
WHERE post_id = ?
ORDER BY created_at ASC;

-- piko.name: CreateComment
-- piko.command: exec
INSERT INTO comments (post_id, author_name, body, created_at)
VALUES (?, ?, ?, ?);

-- piko.name: GetCommentCount
-- piko.command: one
SELECT COUNT(*) as total FROM comments WHERE post_id = ?;
