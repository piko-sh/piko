-- piko.query(name: FindByMatch, command: many)
SELECT id, name FROM files WHERE path MATCH ?
