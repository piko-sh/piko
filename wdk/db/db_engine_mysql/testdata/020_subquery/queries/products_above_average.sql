-- piko.name: ProductsAboveAverage
-- piko.command: many
SELECT id, name, price
FROM products
WHERE price > (SELECT AVG(price) FROM products);
