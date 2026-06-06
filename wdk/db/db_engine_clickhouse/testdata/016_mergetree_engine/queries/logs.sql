-- piko.query(LookupLog, one)
SELECT id, ts, level, message FROM logs WHERE id = {id:UInt64};
