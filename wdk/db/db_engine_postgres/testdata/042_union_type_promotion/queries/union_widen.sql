-- piko.name: UnionWiden
-- piko.command: many
SELECT val FROM narrow
UNION ALL
SELECT val FROM wide;
