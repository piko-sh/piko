-- piko.query(InsertEpoch, exec)
INSERT INTO epochs (id, day) VALUES ({id:UInt64}, {day:Date32});

-- piko.query(List, many)
SELECT id, day FROM epochs ORDER BY id;
