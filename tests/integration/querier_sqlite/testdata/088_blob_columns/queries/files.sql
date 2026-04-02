-- piko.name: InsertFile
-- piko.command: exec
INSERT INTO files (id, name, content, size) VALUES (?, ?, ?, ?);

-- piko.name: GetFile
-- piko.command: one
SELECT id, name, content, size FROM files WHERE id = ?;

-- piko.name: ListFileNames
-- piko.command: many
SELECT id, name, size FROM files ORDER BY id;
