-- piko.query(InsertKeyed, exec)
INSERT INTO keyed (id, payload, version) VALUES ({id:UInt64}, {payload:String}, {version:UInt64});

-- piko.query(Deduped, many)
SELECT id, payload, version FROM keyed FINAL ORDER BY id;
