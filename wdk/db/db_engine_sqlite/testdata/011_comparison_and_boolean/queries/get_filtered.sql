-- piko.name: GetFilteredTasks
-- piko.command: many
SELECT
  id,
  priority > 5 as is_high,
  assigned_to IS NOT NULL as is_assigned,
  completed AND priority > 3 as important_done
FROM tasks
