-- piko.name: GetDepartmentSummary
-- piko.command: many
SELECT
    d.id,
    d.name,
    (SELECT COUNT(*) FROM employees e WHERE e.department_id = d.id) AS employee_count,
    (SELECT MAX(e.salary) FROM employees e WHERE e.department_id = d.id) AS max_salary
FROM departments d
ORDER BY d.id;
