-- piko.query(GroupedVals, many)
SELECT cat, groupArray(val) AS vals FROM t GROUP BY cat;
