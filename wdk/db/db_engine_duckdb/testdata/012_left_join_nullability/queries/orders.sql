-- piko.query(name: GetOrdersWithCustomerName, command: many)
SELECT o.id, c.name FROM orders o LEFT JOIN customers c ON o.customer_id = c.id;
