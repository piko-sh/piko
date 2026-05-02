-- piko.query(InsertAddress, exec)
INSERT INTO addresses (id, addr) VALUES ({id:UInt64}, {addr:IPv4});

-- piko.query(List, many)
SELECT id, addr FROM addresses ORDER BY id;
