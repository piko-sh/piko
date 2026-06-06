-- piko.query(InsertSample, exec)
INSERT INTO samples (cohort, value) VALUES ({cohort:String}, {value:Float64});

-- piko.query(WelchTest, one)
SELECT
    tupleElement(result, 1) AS t_statistic,
    tupleElement(result, 2) AS p_value
FROM (
    SELECT welchTTest(value, cohort = 'b') AS result
    FROM samples
);
