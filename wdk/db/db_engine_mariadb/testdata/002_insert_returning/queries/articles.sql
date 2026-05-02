-- piko.query(name: CreateArticle, command: one)
INSERT INTO articles (title, body) VALUES (?, ?) RETURNING id, title;

-- piko.query(name: GetArticle, command: one)
SELECT id, title, body, published_at FROM articles WHERE id = ?;
