-- piko.query(name: UnionWiden, command: many)
SELECT val FROM narrow
UNION ALL
SELECT val FROM wide;
