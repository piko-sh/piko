-- piko.name: RandomItems
-- piko.command: many
SELECT id, name FROM items ORDER BY RANDOM() LIMIT $1;
