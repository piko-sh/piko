-- piko.name: DeptEmployees
-- piko.command: many
SELECT d.name AS dept_name, e.name AS employee_name
FROM departments d
LEFT JOIN employees e ON e.dept_id = d.id;
