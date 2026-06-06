-- piko.query(InsertProduct, exec)
INSERT INTO products (id, name) VALUES ({id:UInt64}, {name:String});

-- piko.query(InsertReview, exec)
INSERT INTO reviews (product_id, stars) VALUES ({product_id:UInt64}, {stars:UInt8});

-- piko.query(WithReviews, many)
SELECT p.id, p.name, r.stars
FROM products AS p
LEFT JOIN reviews AS r ON r.product_id = p.id
ORDER BY p.id, r.stars;
