-- piko.query(Decoded, many)
SELECT id, JSONExtractString(body, 'name') AS name, JSONExtractInt(body, 'age') AS age FROM payloads ORDER BY id;
