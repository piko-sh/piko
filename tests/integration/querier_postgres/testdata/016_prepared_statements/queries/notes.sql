-- piko.name: InsertNote
-- piko.command: exec
INSERT INTO notes (title, body) VALUES ($1, $2);

-- piko.name: GetNote
-- piko.command: one
SELECT id, title, body FROM notes WHERE id = $1;

-- piko.name: ListNotes
-- piko.command: many
SELECT id, title, body FROM notes ORDER BY id;
