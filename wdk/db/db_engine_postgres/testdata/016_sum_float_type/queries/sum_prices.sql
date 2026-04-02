-- piko.name: SumPrices
-- piko.command: one
SELECT SUM(price) AS total_price, SUM(discount) AS total_discount FROM products;
