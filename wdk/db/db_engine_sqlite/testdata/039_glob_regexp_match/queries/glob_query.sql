-- piko.query(name: FindByGlob, command: many)
SELECT id, name FROM files WHERE name GLOB ?
