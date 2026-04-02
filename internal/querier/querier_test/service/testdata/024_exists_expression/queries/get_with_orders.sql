-- piko.name: GetUsersWithOrders
-- piko.command: many
SELECT
  id,
  name,
  EXISTS(SELECT 1 FROM orders WHERE orders.user_id = users.id) as has_orders
FROM users
