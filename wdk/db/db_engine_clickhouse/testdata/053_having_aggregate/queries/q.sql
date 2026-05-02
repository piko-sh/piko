-- piko.query(BigCats, many)
SELECT cat, sum(val) AS s FROM t GROUP BY cat HAVING s > {threshold:UInt64};
