-- piko.query(LatestPerCat, many)
SELECT id, cat, ts FROM t ORDER BY cat, ts DESC LIMIT 3 BY cat;
