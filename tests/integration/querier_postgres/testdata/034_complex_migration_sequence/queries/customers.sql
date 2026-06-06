-- piko.query(name: GetCustomerByEmail, command: one)
SELECT id, name, email, tier
FROM customers
WHERE email = $1;

-- piko.query(name: GetOrderSummary, command: many)
SELECT customer_id, name, email, order_count, total_spent
FROM customer_order_summary
ORDER BY total_spent DESC;

-- piko.query(name: InsertOrder, command: one)
INSERT INTO orders (customer_id, total, status, notes)
VALUES ($1, $2, $3, $4)
RETURNING id, customer_id, total, status, notes;
