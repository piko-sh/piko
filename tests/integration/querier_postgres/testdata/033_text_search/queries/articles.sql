-- piko.name: SearchArticles
-- piko.command: many
SELECT id, title,
    ts_rank(search_vector, to_tsquery('english', $1)) AS rank
FROM articles
WHERE search_vector @@ to_tsquery('english', $1)
ORDER BY rank DESC, id;

-- piko.name: SearchWithHeadline
-- piko.command: many
SELECT id, title,
    ts_headline('english', body, to_tsquery('english', $1), 'MaxWords=20, MinWords=10') AS headline
FROM articles
WHERE search_vector @@ to_tsquery('english', $1)
ORDER BY id;
