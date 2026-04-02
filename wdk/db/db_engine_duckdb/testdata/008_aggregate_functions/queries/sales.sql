-- piko.name: GetSalesSummary
-- piko.command: one
SELECT count(*) AS total_count, sum(amount) AS total_amount, avg(amount) AS avg_amount, min(amount) AS min_amount, max(amount) AS max_amount FROM sales;
