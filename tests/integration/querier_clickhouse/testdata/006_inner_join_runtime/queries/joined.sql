-- piko.query(InsertUser, exec)
INSERT INTO users (id, name) VALUES ({id:UInt64}, {name:String});

-- piko.query(InsertOrder, exec)
INSERT INTO orders (id, user_id, amount) VALUES ({id:UInt64}, {user_id:UInt64}, {amount:UInt32});

-- piko.query(ListOrdersWithUsers, many)
SELECT o.id AS order_id, u.name AS user_name, o.amount AS amount
FROM orders AS o
INNER JOIN users AS u ON u.id = o.user_id
ORDER BY o.id;
