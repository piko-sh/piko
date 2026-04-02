-- piko.name: WidenParam
-- piko.command: many
SELECT t1.a, t2.b FROM t1, t2 WHERE t1.a = $1 AND t2.b = $1;
