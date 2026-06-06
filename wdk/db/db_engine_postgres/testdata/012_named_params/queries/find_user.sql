-- piko.query(name: FindUser, command: one)
SELECT id, name, email FROM users WHERE name = :name AND age > :min_age;
