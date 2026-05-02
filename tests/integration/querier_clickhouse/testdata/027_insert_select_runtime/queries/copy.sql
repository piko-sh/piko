-- piko.query(InsertSource, exec)
INSERT INTO source (id, val) VALUES ({id:UInt64}, {val:UInt64});

-- piko.query(CopySourceToDest, exec)
INSERT INTO dest (id, val) SELECT id, val * 10 FROM source;

-- piko.query(ReadDest, many)
SELECT id, val FROM dest ORDER BY id;
