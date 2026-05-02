-- piko.query(name: ListUserOrders, command: many)
-- $1 as piko.param(userId)
-- $2 as piko.param(status, optional: true)
-- $3 as piko.param(pageSize)
-- $4 as piko.param(pageOffset)
SELECT u.name, o.total, o.status
FROM users u
JOIN orders o ON o.user_id = u.id
WHERE u.id = $1
  AND ($2 IS NULL OR o.status = $2)
LIMIT $3 OFFSET $4
