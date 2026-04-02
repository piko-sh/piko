-- piko.name: TopEarnerPerDepartment
-- piko.command: many
SELECT d.name AS department_name, e.name AS employee_name, e.salary
FROM departments d,
LATERAL (
    SELECT name, salary
    FROM employees
    WHERE dept_id = d.id
    ORDER BY salary DESC
    LIMIT 1
) e
ORDER BY d.name;

-- piko.name: TopTwoPerDepartment
-- piko.command: many
SELECT d.name AS department_name, e.name AS employee_name, e.salary
FROM departments d,
LATERAL (
    SELECT name, salary
    FROM employees
    WHERE dept_id = d.id
    ORDER BY salary DESC
    LIMIT 2
) e
ORDER BY d.name, e.salary DESC;
