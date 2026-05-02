-- piko.query(WithMax, many)
SELECT id, val FROM prices WHERE val = (SELECT max(val) FROM prices);
