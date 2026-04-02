-- piko.name: FetchByStatusesSorted
-- piko.command: many
-- ?1 as piko.slice(statuses)
-- ?2 as piko.sortable(order_by) columns:id,priority,status
SELECT id, status, priority, title
FROM tasks
WHERE status IN (?1)
