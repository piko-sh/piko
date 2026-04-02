-- piko.name: GetProductsWithMaxPrice
-- piko.command: many
SELECT name, price, (SELECT max(price) FROM products WHERE category = ?) AS max_price FROM products WHERE category = ?
