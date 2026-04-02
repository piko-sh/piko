-- piko.name: CreateArticle
-- piko.command: one
INSERT INTO articles (title, body) VALUES (?, ?) RETURNING id, title;

-- piko.name: GetArticle
-- piko.command: one
SELECT id, title, body, published_at FROM articles WHERE id = ?;
