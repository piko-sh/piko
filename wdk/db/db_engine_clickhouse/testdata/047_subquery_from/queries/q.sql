-- piko.query(DerivedSelect, many)
SELECT s.id, s.x FROM (SELECT id, x FROM t WHERE x > 0) s;
