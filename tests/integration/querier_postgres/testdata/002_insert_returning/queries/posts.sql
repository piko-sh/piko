-- piko.name: CreatePost
-- piko.command: one
INSERT INTO posts (title, body, author_id, published)
VALUES ($1, $2, $3, $4)
RETURNING id, title, published;
