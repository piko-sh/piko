-- piko.name: AuthorsWithBooks
-- piko.command: many
SELECT a.id, a.name FROM authors a
WHERE EXISTS (SELECT 1 FROM books b WHERE b.author_id = a.id)
ORDER BY a.name;
