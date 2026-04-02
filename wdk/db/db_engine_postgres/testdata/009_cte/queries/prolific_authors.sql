-- piko.name: ProlificAuthors
-- piko.command: many
WITH book_counts AS (
    SELECT author_id, COUNT(*) AS book_count
    FROM books
    GROUP BY author_id
)
SELECT a.name, bc.book_count
FROM authors a
JOIN book_counts bc ON bc.author_id = a.id
WHERE bc.book_count > $1
ORDER BY bc.book_count DESC;
