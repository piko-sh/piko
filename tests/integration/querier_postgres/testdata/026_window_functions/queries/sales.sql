-- piko.query(name: RankByDepartment, command: many)
SELECT id, department, employee, amount,
    ROW_NUMBER() OVER (PARTITION BY department ORDER BY amount DESC) AS row_num,
    RANK() OVER (PARTITION BY department ORDER BY amount DESC) AS rank,
    DENSE_RANK() OVER (PARTITION BY department ORDER BY amount DESC) AS dense_rank
FROM sales
ORDER BY department, amount DESC;

-- piko.query(name: LagLeadAnalysis, command: many)
SELECT id, employee, amount,
    LAG(amount, 1) OVER (ORDER BY id) AS prev_amount,
    LEAD(amount, 1) OVER (ORDER BY id) AS next_amount
FROM sales
ORDER BY id;

-- piko.query(name: RunningTotalByDepartment, command: many)
SELECT id, department, employee, amount,
    SUM(amount) OVER (PARTITION BY department ORDER BY id) AS running_total
FROM sales
ORDER BY department, id;
