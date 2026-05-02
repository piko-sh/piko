-- piko.query(Insert, exec)
INSERT INTO events (id, country) VALUES ({id:UInt64}, {country:String});

-- piko.query(List, many)
SELECT id, country FROM events ORDER BY id;
