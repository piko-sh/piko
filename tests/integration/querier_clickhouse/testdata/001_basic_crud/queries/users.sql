-- piko.query(InsertUser, exec)
INSERT INTO users (id, name) VALUES ({id:UInt64}, {name:String});

-- piko.query(GetUser, one)
SELECT id, name FROM users WHERE id = {id:UInt64};

-- piko.query(ListUsers, many)
SELECT id, name FROM users ORDER BY id;
