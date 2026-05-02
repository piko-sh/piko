-- piko.query(name: GetOrderCount, command: one)
SELECT order_count() AS total_orders;

-- piko.query(name: GetLatestCustomer, command: one)
SELECT latest_customer() AS customer;
