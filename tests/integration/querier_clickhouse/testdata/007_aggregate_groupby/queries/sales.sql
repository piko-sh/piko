-- piko.query(InsertSale, exec)
INSERT INTO sales (id, category, amount) VALUES ({id:UInt64}, {category:String}, {amount:UInt32});

-- piko.query(SalesByCategory, many)
SELECT category, sum(amount) AS total, count() AS sales_count
FROM sales
GROUP BY category
ORDER BY category;
