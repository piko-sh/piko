-- piko.query(name: GenerateToken, command: one)
SELECT UUID() AS token_value;

-- piko.query(name: InsertToken, command: exec)
INSERT INTO tokens (token) VALUES (?);
