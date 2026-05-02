-- piko.query(name: MostExpensivePerCategory, command: many)
SELECT DISTINCT ON (category) id, name, category, price
FROM products
ORDER BY category, price DESC;

-- piko.query(name: CheapestPerCategory, command: many)
SELECT DISTINCT ON (category) id, name, category, price
FROM products
ORDER BY category, price ASC;
