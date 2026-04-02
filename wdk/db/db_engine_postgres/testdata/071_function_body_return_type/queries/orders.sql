-- piko.name: GetOrderCount
-- piko.command: one
SELECT order_count() AS total_orders;

-- piko.name: GetLatestCustomer
-- piko.command: one
SELECT latest_customer() AS customer;
