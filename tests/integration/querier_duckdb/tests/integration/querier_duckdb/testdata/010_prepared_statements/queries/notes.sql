-- piko.query(name: InsertNote, command: exec)
INSERT INTO notes (id, title, body) VALUES ($1, $2, $3);

-- piko.query(name: GetNote, command: one)
SELECT id, title, body FROM notes WHERE id = $1;

-- piko.query(name: ListNotes, command: many)
SELECT id, title, body FROM notes ORDER BY id;
