-- piko.query(InsertEvent, exec)
INSERT INTO events (id, kind) VALUES ({id:UInt64}, {kind:String});

-- piko.query(KindCounts, many)
WITH counts AS (SELECT kind, count(*) AS c FROM events GROUP BY kind)
SELECT kind, c FROM counts ORDER BY c DESC;
