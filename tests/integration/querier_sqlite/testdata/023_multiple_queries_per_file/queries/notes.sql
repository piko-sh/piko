-- piko.query(name: CreateNote, command: one)
INSERT INTO notes (title, body) VALUES (?, ?) RETURNING id, title, body;

-- piko.query(name: GetNote, command: one)
SELECT id, title, body FROM notes WHERE id = ?;

-- piko.query(name: ListNotes, command: many)
SELECT id, title, body FROM notes ORDER BY id;

-- piko.query(name: UpdateNoteTitle, command: execrows)
UPDATE notes SET title = ? WHERE id = ?;

-- piko.query(name: DeleteNote, command: exec)
DELETE FROM notes WHERE id = ?;
