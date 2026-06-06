-- piko.query(InsertDocument, exec)
INSERT INTO documents (id, body) VALUES ({id:UInt64}, {body:String});

-- piko.query(JSONPathFields, many)
SELECT
    id,
    JSON_VALUE(body, '$.user.name')          AS user_name,
    JSON_QUERY(body, '$.user')               AS user_sub,
    JSON_EXISTS(body, '$.user.email')        AS has_email
FROM documents
ORDER BY id;
