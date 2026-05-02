-- piko.query(InsertSale, exec)
INSERT INTO sales (id, region, amount) VALUES ({id:UInt64}, {region:String}, {amount:UInt64});

-- piko.query(TotalByRegion, many)
SELECT region, sum(amount) AS total FROM sales GROUP BY region ORDER BY total DESC;
