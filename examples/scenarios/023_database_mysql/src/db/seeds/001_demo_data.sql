-- Seed demo data for the blog example. Uses ON DUPLICATE KEY UPDATE to ensure
-- idempotency when the seed is re-applied after a reseed.

INSERT INTO authors (name, email, created_at)
VALUES ('Alice Williams', 'alice@example.com', UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE name = name;

INSERT INTO authors (name, email, created_at)
VALUES ('Bob Smith', 'bob@example.com', UNIX_TIMESTAMP())
ON DUPLICATE KEY UPDATE name = name;

INSERT INTO posts (author_id, title, body, published, created_at)
SELECT a.id,
       'Welcome to the Blog',
       'This is a demo blog built with Piko and MySQL. Try creating new posts and adding comments!',
       TRUE,
       UNIX_TIMESTAMP()
FROM authors a
WHERE a.email = 'alice@example.com'
  AND NOT EXISTS (SELECT 1 FROM posts WHERE title = 'Welcome to the Blog');

INSERT INTO comments (post_id, author_name, body, created_at)
SELECT p.id,
       'Bob Smith',
       'Great first post!',
       UNIX_TIMESTAMP()
FROM posts p
WHERE p.title = 'Welcome to the Blog'
  AND NOT EXISTS (SELECT 1 FROM comments WHERE post_id = p.id AND author_name = 'Bob Smith');
