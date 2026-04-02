-- piko.name: GetEmployeeWithManager
-- piko.command: many
SELECT e.id, e.name AS employee_name, m.name AS manager_name
FROM employees e
LEFT JOIN employees m ON m.id = e.manager_id
ORDER BY e.id;
