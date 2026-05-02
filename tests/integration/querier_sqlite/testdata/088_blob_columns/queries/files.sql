-- piko.query(name: InsertFile, command: exec)
INSERT INTO files (id, name, content, size) VALUES (?, ?, ?, ?);

-- piko.query(name: GetFile, command: one)
SELECT id, name, content, size FROM files WHERE id = ?;

-- piko.query(name: ListFileNames, command: many)
SELECT id, name, size FROM files ORDER BY id;
