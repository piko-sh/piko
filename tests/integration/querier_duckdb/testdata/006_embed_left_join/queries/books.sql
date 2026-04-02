-- piko.name: GetBookWithReview
-- piko.command: one
SELECT b.id, b.title, /* piko.embed(reviews) */ r.id, r.rating
FROM books b
LEFT JOIN reviews r ON r.book_id = b.id
WHERE b.id = $1;
