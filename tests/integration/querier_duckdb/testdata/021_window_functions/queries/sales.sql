-- piko.query(name: ListWithRunningTotal, command: many)
SELECT id, employee, amount, SUM(amount) OVER (ORDER BY id) AS running_total FROM sales ORDER BY id;

-- piko.query(name: ListWithLagLead, command: many)
SELECT id, employee, amount, LAG(amount, 1) OVER (ORDER BY id) AS prev_amount, LEAD(amount, 1) OVER (ORDER BY id) AS next_amount FROM sales ORDER BY id;

-- piko.query(name: TopSalePerEmployee, command: many)
SELECT id, employee, amount FROM sales QUALIFY RANK() OVER (PARTITION BY employee ORDER BY amount DESC) = 1 ORDER BY employee;
