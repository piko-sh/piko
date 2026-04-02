-- piko.name: CreateToken
-- piko.command: one
INSERT INTO tokens (value) VALUES ($1) RETURNING id, value;
