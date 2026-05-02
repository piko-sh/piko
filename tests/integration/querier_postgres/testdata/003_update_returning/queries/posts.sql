-- piko.query(name: PublishPost, command: one)
UPDATE posts SET published = TRUE WHERE id = $1 RETURNING id, title, published;
