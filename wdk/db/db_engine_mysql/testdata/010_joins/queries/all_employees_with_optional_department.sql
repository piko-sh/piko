-- piko.name: AllEmployeesWithOptionalDepartment
-- piko.command: many
SELECT e.id, e.name, d.name AS department_name
FROM employees e
LEFT JOIN departments d ON e.department_id = d.id;
