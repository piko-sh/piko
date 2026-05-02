-- piko.query(name: InsertArticle, command: exec)
INSERT INTO articles (title, body, author, published_at) VALUES (?, ?, ?, ?);

-- piko.query(name: GetByAuthor, command: many)
SELECT id, title, author FROM articles WHERE author = ? ORDER BY id;

-- piko.query(name: GetByTitle, command: one)
SELECT id, title, author FROM articles WHERE title = ?;

-- piko.query(name: FulltextSearch, command: many)
SELECT id, title, author FROM articles WHERE MATCH(title, body) AGAINST(? IN BOOLEAN MODE) ORDER BY id;
