-- piko.query(name: RandomItems, command: many)
SELECT id, name FROM items ORDER BY RANDOM() LIMIT $1;
