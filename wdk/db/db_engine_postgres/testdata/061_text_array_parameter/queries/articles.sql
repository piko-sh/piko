-- piko.query(name: GetArticlesByTag, command: many)
SELECT id, title, tags FROM articles WHERE $1::text = ANY(tags);

-- piko.query(name: GetArticlesWithTags, command: many)
SELECT id, title, tags FROM articles WHERE tags @> $1::text[];
