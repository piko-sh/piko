-- piko.query(name: DocumentsByCategory, command: many)
SELECT id, title, category, created_at FROM documents WHERE category = ?;
