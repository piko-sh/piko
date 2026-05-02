-- piko.query(name: LatestPerCategory, command: many)
SELECT DISTINCT ON (category) category, tags
FROM events
ORDER BY category, id DESC;
