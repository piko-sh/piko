-- piko.name: SearchArticles
-- piko.command: many
SELECT highlight(articles_fts, 0, '<b>', '</b>') AS title_highlighted,
       snippet(articles_fts, 1, '<b>', '</b>', '...', 32) AS body_snippet,
       bm25(articles_fts) AS relevance
FROM articles_fts WHERE articles_fts MATCH ?
