-- piko.query(UniqueCounts, one)
SELECT uniq(user_id) AS approx_unique, uniqExact(user_id) AS exact_unique FROM t;
