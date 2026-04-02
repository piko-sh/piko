-- piko.name: GetTopLevelEmployees
-- piko.command: many
WITH top_level AS (
    SELECT id, name, manager_id FROM employees WHERE manager_id IS NULL
)
SELECT id, name, manager_id FROM top_level;

-- piko.name: GetEmployeeHierarchy
-- piko.command: many
WITH RECURSIVE hierarchy AS (
    SELECT id, name, manager_id, 0 AS depth FROM employees WHERE manager_id IS NULL
    UNION ALL
    SELECT e.id, e.name, e.manager_id, h.depth + 1 FROM employees e INNER JOIN hierarchy h ON e.manager_id = h.id
)
SELECT id, name, manager_id, depth FROM hierarchy;
