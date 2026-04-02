-- piko.name: GetSalesSummary
-- piko.command: one
SELECT
    count(*) AS total_count,
    sum(amount) AS total_amount,
    avg(amount) AS average_amount,
    min(amount) AS min_amount,
    max(amount) AS max_amount
FROM sales;

-- piko.name: GetSalesByProduct
-- piko.command: many
SELECT
    product,
    count(*) AS sale_count,
    sum(amount) AS total_amount
FROM sales
GROUP BY product
ORDER BY product;
