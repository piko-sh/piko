-- piko.query(name: DeletePost, command: one)
DELETE FROM posts WHERE id = $1 RETURNING id, title;
