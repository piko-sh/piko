ALTER TABLE articles ADD FULLTEXT INDEX idx_articles_fulltext (title, body);
