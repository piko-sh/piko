-- piko.name: InsertArticle
-- piko.command: one
INSERT INTO articles (title, tags, scores)
VALUES ($1, $2::TEXT[], $3::INTEGER[])
RETURNING id, title, tags, scores;

-- piko.name: FilterByTag
-- piko.command: many
SELECT id, title, tags
FROM articles
WHERE $1 = ANY(tags)
ORDER BY id;

-- piko.name: AggregateTagsByTitle
-- piko.command: many
SELECT title, array_agg(DISTINCT unnested_tag ORDER BY unnested_tag) AS all_tags
FROM articles, unnest(tags) AS unnested_tag
GROUP BY title
ORDER BY title;
