-- piko.name: LatestBooks
-- piko.command: many
SELECT a.name, latest.title, latest.published
FROM authors a
LEFT JOIN LATERAL (
    SELECT b.title, b.published
    FROM books b
    WHERE b.author_id = a.id
    ORDER BY b.published DESC
    LIMIT 1
) latest ON TRUE;
