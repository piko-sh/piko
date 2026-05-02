-- piko.query(name: ListWithRunningTotal, command: many)
SELECT id, employee, amount, SUM(amount) OVER (ORDER BY id) AS running_total FROM sales ORDER BY id;

-- piko.query(name: ListWithLagLead, command: many)
SELECT id, employee, amount, LAG(amount, 1) OVER (ORDER BY id) AS prev_amount, LEAD(amount, 1) OVER (ORDER BY id) AS next_amount FROM sales ORDER BY id;

-- piko.query(name: ListWithRankByEmployee, command: many)
SELECT id, employee, amount, RANK() OVER (PARTITION BY employee ORDER BY amount DESC) AS amount_rank FROM sales ORDER BY employee, amount_rank;

-- piko.query(name: ListWithNtile, command: many)
SELECT id, amount, NTILE(3) OVER (ORDER BY amount) AS quartile FROM sales ORDER BY amount;
