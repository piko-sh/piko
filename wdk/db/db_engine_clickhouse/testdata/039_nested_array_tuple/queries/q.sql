-- piko.query(ReadNested, one)
SELECT id, items FROM nested_tbl WHERE id = {id:UInt64};
