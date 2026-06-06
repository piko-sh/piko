-- piko.query(RollupSums, many)
SELECT region, city, sum(val) AS s FROM t GROUP BY region, city WITH ROLLUP;
