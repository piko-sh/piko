-- piko.name: GetProfile
-- piko.command: one
SELECT id, name, biography, age FROM profiles WHERE id = ?;

-- piko.name: ListProfiles
-- piko.command: many
SELECT id, name, biography, age FROM profiles ORDER BY id;
