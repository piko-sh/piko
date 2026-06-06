-- piko.query(Aliased, many)
SELECT inner.id FROM (SELECT id FROM t) AS inner;
