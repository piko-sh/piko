-- piko.name: PublishPost
-- piko.command: one
UPDATE posts SET published = TRUE WHERE id = $1 RETURNING id, title, published;
