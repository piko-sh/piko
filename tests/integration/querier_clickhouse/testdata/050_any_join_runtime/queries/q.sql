-- piko.query(InsertCustomer, exec)
INSERT INTO customers (id, name) VALUES ({id:UInt64}, {name:String});

-- piko.query(InsertOrder, exec)
INSERT INTO orders (id, customer_id, amount) VALUES ({id:UInt64}, {customer_id:UInt64}, {amount:UInt32});

-- piko.query(CustomersWithFirstOrder, many)
SELECT c.id AS customer_id, c.name AS customer_name, o.amount AS amount
FROM customers AS c
ANY LEFT JOIN orders AS o ON o.customer_id = c.id
ORDER BY c.id;
