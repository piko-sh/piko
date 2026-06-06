-- piko.query(name: FetchByStatusesSorted, command: many)
-- ?1 as piko.param(statuses, kind: slice)
-- piko.sortable(order_by, columns: [id, priority, status])
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1)
