-- piko.query(InsertUUID, exec)
INSERT INTO uuid_log (id, label) VALUES ({id:UUID}, {label:String});

-- piko.query(ExtractTimestamps, many)
SELECT
    label,
    UUIDv7ToDateTime(id) AS ts
FROM uuid_log
ORDER BY label;
