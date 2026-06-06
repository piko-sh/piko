-- piko.query(MedianVal, one)
SELECT quantile(0.5)(val) AS median FROM t;
