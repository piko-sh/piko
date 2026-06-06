-- piko.query(name: GetArticle, command: one)
SELECT id, title, status, tags, created_at FROM articles WHERE id = ?;
