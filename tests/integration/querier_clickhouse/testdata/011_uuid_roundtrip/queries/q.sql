-- piko.query(Insert, exec)
INSERT INTO sessions (id, label) VALUES ({id:UUID}, {label:String});

-- piko.query(Get, one)
SELECT id, label FROM sessions WHERE id = {id:UUID};
