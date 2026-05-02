-- piko.query(InsertWith, exec)
INSERT INTO accounts (id, label) VALUES ({id:UInt64}, {label:Nullable(String)});

-- piko.query(List, many)
SELECT id, label FROM accounts ORDER BY id;
