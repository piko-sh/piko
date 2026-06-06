-- piko.query(name: GetDoc, command: one)
SELECT id, payload FROM docs WHERE id = $1;
