-- piko.query(InsertProduct, exec)
INSERT INTO products (id, name, price, in_stock) VALUES ({id:UInt64}, {name:String}, {price:UInt32}, {in_stock:Bool});

-- piko.query(FilterProducts, many)
SELECT id, name, price, in_stock
FROM products
WHERE name = {name:String}
  AND price >= {min_price:UInt32}
  AND in_stock = {only_stocked:Bool}
ORDER BY id;
