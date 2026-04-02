-- piko.name: GetArticle
-- piko.command: one
SELECT id, title, status, tags, created_at FROM articles WHERE id = ?;
