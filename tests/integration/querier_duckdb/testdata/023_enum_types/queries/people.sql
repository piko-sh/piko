-- piko.name: InsertPerson
-- piko.command: exec
INSERT INTO people VALUES ($1, $2, $3);

-- piko.name: GetPerson
-- piko.command: one
SELECT id, name, current_mood FROM people WHERE id = $1;

-- piko.name: ListByMood
-- piko.command: many
SELECT id, name FROM people WHERE current_mood = $1 ORDER BY id;
