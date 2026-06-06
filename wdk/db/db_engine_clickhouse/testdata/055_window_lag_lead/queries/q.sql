-- piko.query(Sequential, many)
SELECT id, val, lag(val) OVER (ORDER BY ts) AS prev_val FROM t;
