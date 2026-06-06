-- piko.query(InsertCoord, exec)
INSERT INTO coords (id, position) VALUES ({id:UInt64}, ({pos_name:String}, {pos_value:UInt32}));

-- piko.query(List, many)
SELECT id, position FROM coords ORDER BY id;
