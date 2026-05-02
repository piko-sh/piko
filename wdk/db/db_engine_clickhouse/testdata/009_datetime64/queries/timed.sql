-- piko.query(GetTimed, one)
SELECT id, micro_ts, nano_ts FROM timed WHERE id = {id:UInt64};
