-- piko.query(name: GetProfile, command: one)
SELECT id, name, biography, age FROM profiles WHERE id = ?;

-- piko.query(name: ListProfiles, command: many)
SELECT id, name, biography, age FROM profiles ORDER BY id;
