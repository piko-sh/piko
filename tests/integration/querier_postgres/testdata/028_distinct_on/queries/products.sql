-- piko.name: MostExpensivePerCategory
-- piko.command: many
SELECT DISTINCT ON (category) id, name, category, price
FROM products
ORDER BY category, price DESC;

-- piko.name: CheapestPerCategory
-- piko.command: many
SELECT DISTINCT ON (category) id, name, category, price
FROM products
ORDER BY category, price ASC;
