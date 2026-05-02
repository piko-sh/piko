-- piko.query(InsertPayload, exec)
INSERT INTO payloads (id, body) VALUES ({id:UInt64}, {body:String});

-- piko.query(Hashes, many)
SELECT
    id,
    xxh3(body) AS hash64
FROM payloads
ORDER BY id;
