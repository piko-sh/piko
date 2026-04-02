-- piko.name: FindByMatch
-- piko.command: many
SELECT id, name FROM files WHERE path MATCH ?
