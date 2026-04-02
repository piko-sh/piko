-- piko.name: GetDocument
-- piko.command: one
SELECT id, title, content, checksum FROM documents WHERE id = $1;

-- piko.name: ListDocuments
-- piko.command: many
SELECT id, title FROM documents ORDER BY title;
