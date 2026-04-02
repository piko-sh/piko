-- piko.name: CreateNote
-- piko.command: one
INSERT INTO notes (title, body) VALUES (?, ?) RETURNING id, title, body;

-- piko.name: GetNote
-- piko.command: one
SELECT id, title, body FROM notes WHERE id = ?;

-- piko.name: ListNotes
-- piko.command: many
SELECT id, title, body FROM notes ORDER BY id;

-- piko.name: UpdateNoteTitle
-- piko.command: execrows
UPDATE notes SET title = ? WHERE id = ?;

-- piko.name: DeleteNote
-- piko.command: exec
DELETE FROM notes WHERE id = ?;
