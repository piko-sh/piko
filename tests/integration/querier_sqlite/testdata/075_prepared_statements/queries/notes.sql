-- piko.name: InsertNote
-- piko.command: exec
INSERT INTO notes (id, title, body) VALUES (?, ?, ?);

-- piko.name: GetNote
-- piko.command: one
SELECT id, title, body FROM notes WHERE id = ?;

-- piko.name: ListNotes
-- piko.command: many
SELECT id, title, body FROM notes ORDER BY id;
