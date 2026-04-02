-- piko.name: ListEmployeesWithDepartment
-- piko.command: many
SELECT e.id, e.name, d.name AS department_name
FROM employees e
LEFT JOIN departments d ON d.id = e.department_id
ORDER BY e.id;
