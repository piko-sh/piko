-- piko.query(SumByCat, many)
SELECT cat, sum(val) AS s FROM t GROUP BY cat WITH TOTALS;
