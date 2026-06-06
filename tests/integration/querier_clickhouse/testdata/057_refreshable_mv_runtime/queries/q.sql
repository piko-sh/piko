-- piko.query(InsertEvent, exec)
INSERT INTO source_events (id, value) VALUES ({id:UInt64}, {value:UInt64});

-- piko.query(ReadTarget, one)
SELECT total
FROM refresh_target
ORDER BY total DESC
LIMIT 1;
