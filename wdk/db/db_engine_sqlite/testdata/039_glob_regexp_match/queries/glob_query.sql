-- piko.name: FindByGlob
-- piko.command: many
SELECT id, name FROM files WHERE name GLOB ?
