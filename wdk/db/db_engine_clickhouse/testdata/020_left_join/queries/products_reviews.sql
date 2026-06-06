-- piko.query(GetProductReview, one)
SELECT p.id, p.name, r.stars
FROM products p LEFT JOIN reviews r ON r.product_id = p.id
WHERE p.id = {pid:UInt64};
