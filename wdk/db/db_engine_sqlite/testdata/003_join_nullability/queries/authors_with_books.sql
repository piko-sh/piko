-- piko.name: ListAuthorsWithBooks
-- piko.command: many
SELECT a.id, a.name, b.id, b.title
FROM authors a
LEFT JOIN books b ON b.author_id = a.id
