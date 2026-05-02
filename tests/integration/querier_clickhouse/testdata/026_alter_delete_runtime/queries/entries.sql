-- piko.query(InsertEntry, exec)
INSERT INTO entries (id, tombstoned) VALUES ({id:UInt64}, {tombstoned:UInt8});

-- piko.query(TombstoneAll, exec)
ALTER TABLE entries DELETE WHERE id = {id:UInt64};

-- piko.query(Live, many)
SELECT id, tombstoned FROM entries WHERE tombstoned = 0 ORDER BY id;

-- piko.query(AllRows, many)
SELECT id, tombstoned FROM entries ORDER BY id;
