-- piko.query(name: GetBookWithReview, command: one)
-- piko.embed(reviews, from: r)
SELECT b.id, b.title,  r.id, r.rating
FROM books b
LEFT JOIN reviews r ON r.book_id = b.id
WHERE b.id = ?;
