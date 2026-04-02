-- piko.name: GetTaskComparisons
-- piko.command: many
SELECT
  id,
  priority > 5 as is_high_priority,
  score <> 0.0 as has_score
FROM tasks
