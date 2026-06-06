-- piko.query(InsertPayload, exec)
INSERT INTO payloads (id, body) VALUES ({id:UInt64}, {body:String});

-- piko.query(DecodedSimple, many)
SELECT
    id,
    simpleJSONExtractString(body, 'name') AS name,
    simpleJSONExtractInt(body, 'age')     AS age
FROM payloads
ORDER BY id;
