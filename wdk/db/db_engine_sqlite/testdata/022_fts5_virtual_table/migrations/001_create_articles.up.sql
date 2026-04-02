CREATE TABLE articles (
  id INTEGER PRIMARY KEY,
  title TEXT NOT NULL,
  body TEXT NOT NULL
);

CREATE VIRTUAL TABLE articles_fts USING fts5(title, body, content=articles);
