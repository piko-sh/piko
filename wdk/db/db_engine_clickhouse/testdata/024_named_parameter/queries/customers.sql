-- piko.query(GetCustomer, one)
SELECT id, email, country FROM customers WHERE id = {customer_id:UInt64};
