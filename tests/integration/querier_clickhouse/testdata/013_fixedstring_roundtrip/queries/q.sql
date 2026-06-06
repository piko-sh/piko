-- piko.query(Insert, exec)
INSERT INTO hashes (id, sha) VALUES ({id:UInt64}, {sha:FixedString(20)});

-- piko.query(Get, one)
SELECT id, sha FROM hashes WHERE id = {id:UInt64};
