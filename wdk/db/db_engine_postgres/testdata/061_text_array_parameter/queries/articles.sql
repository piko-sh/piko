-- piko.name: GetArticlesByTag
-- piko.command: many
SELECT id, title, tags FROM articles WHERE $1::text = ANY(tags);

-- piko.name: GetArticlesWithTags
-- piko.command: many
SELECT id, title, tags FROM articles WHERE tags @> $1::text[];
