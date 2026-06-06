-- piko.query(InsertPrice, exec)
INSERT INTO prices (id, amount) VALUES ({id:UInt64}, {amount:Decimal128(18)});

-- piko.query(List, many)
SELECT id, amount FROM prices ORDER BY id;
