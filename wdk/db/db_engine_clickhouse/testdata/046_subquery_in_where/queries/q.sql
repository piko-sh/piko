-- piko.query(MaxRecord, one)
SELECT id, val FROM t WHERE val = (SELECT max(val) FROM t);
