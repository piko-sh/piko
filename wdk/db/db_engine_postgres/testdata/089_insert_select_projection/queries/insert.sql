-- piko.query(name: InsertTransformation, command: exec)
INSERT INTO media_transformations (id, source_algorithm, priority)
SELECT $1, $2, $3;
