-- piko.query(ByDay, many)
SELECT id, ts FROM t WHERE ts >= {start:DateTime};
