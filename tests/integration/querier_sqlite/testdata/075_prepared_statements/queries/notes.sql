-- piko.query(name: InsertNote, command: exec)
INSERT INTO notes (id, title, body) VALUES (?, ?, ?);

-- piko.query(name: GetNote, command: one)
SELECT id, title, body FROM notes WHERE id = ?;

-- piko.query(name: ListNotes, command: many)
SELECT id, title, body FROM notes ORDER BY id;
