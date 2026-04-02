-- piko.name: ListWithRunningTotal
-- piko.command: many
SELECT id, employee, amount, SUM(amount) OVER (ORDER BY id) AS running_total FROM sales ORDER BY id;

-- piko.name: ListWithLagLead
-- piko.command: many
SELECT id, employee, amount, LAG(amount, 1) OVER (ORDER BY id) AS prev_amount, LEAD(amount, 1) OVER (ORDER BY id) AS next_amount FROM sales ORDER BY id;

-- piko.name: ListWithRankByEmployee
-- piko.command: many
SELECT id, employee, amount, RANK() OVER (PARTITION BY employee ORDER BY amount DESC) AS amount_rank FROM sales ORDER BY employee, amount_rank;

-- piko.name: ListWithRowNumber
-- piko.command: many
SELECT id, employee, amount, ROW_NUMBER() OVER (ORDER BY amount DESC) AS row_num FROM sales ORDER BY row_num;
