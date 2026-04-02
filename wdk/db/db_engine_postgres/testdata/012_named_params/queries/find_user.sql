-- piko.name: FindUser
-- piko.command: one
SELECT id, name, email FROM users WHERE name = :name AND age > :min_age;
