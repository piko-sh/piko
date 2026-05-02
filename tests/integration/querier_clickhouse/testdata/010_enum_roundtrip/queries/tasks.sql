-- piko.query(InsertTask, exec)
INSERT INTO tasks (id, status) VALUES ({id:UInt64}, {status:String});

-- piko.query(GetTask, one)
SELECT id, status FROM tasks WHERE id = {id:UInt64};

-- piko.query(ListTasks, many)
SELECT id, status FROM tasks ORDER BY id;
