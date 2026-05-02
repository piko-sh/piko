-- piko.query(InsertEvent, exec)
INSERT INTO events (id, tags) VALUES ({id:UInt64}, {tags:Array(String)});

-- piko.query(ByTag, many)
SELECT id, tag FROM events ARRAY JOIN tags AS tag ORDER BY id, tag;
