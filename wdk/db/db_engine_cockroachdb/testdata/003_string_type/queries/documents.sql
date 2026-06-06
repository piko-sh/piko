-- piko.query(name: GetDocument, command: one)
SELECT id, title, content, checksum FROM documents WHERE id = $1;

-- piko.query(name: ListDocuments, command: many)
SELECT id, title FROM documents ORDER BY title;
