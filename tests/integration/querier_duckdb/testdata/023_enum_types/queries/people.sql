-- piko.query(name: InsertPerson, command: exec)
INSERT INTO people VALUES ($1, $2, $3);

-- piko.query(name: GetPerson, command: one)
SELECT id, name, current_mood FROM people WHERE id = $1;

-- piko.query(name: ListByMood, command: many)
SELECT id, name FROM people WHERE current_mood = $1 ORDER BY id;
