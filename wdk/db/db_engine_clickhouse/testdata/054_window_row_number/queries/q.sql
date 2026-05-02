-- piko.query(Ranked, many)
SELECT id, cat, row_number() OVER (PARTITION BY cat ORDER BY val DESC) AS rn FROM t;
