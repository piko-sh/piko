-- piko.name: GenerateToken
-- piko.command: one
SELECT UUID() AS token_value;

-- piko.name: InsertToken
-- piko.command: exec
INSERT INTO tokens (token) VALUES (?);
