-- piko.query(name: GetDoc, command: one)
SELECT id, payload, tags FROM docs WHERE id = $1;
