-- piko.name: InsertArticle
-- piko.command: exec
INSERT INTO articles (title, body, author, published_at) VALUES (?, ?, ?, ?);

-- piko.name: GetByAuthor
-- piko.command: many
SELECT id, title, author FROM articles WHERE author = ? ORDER BY id;

-- piko.name: GetByTitle
-- piko.command: one
SELECT id, title, author FROM articles WHERE title = ?;

-- piko.name: FulltextSearch
-- piko.command: many
SELECT id, title, author FROM articles WHERE MATCH(title, body) AGAINST(? IN BOOLEAN MODE) ORDER BY id;
