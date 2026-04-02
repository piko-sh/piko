-- piko.name: InsertDepartment
-- piko.command: exec
INSERT INTO departments (id, name) VALUES (?, ?);

-- piko.name: InsertEmployee
-- piko.command: exec
INSERT INTO employees (id, name, dept_id) VALUES (?, ?, ?);

-- piko.name: ListEmployees
-- piko.command: many
SELECT id, name, dept_id FROM employees ORDER BY id;

-- piko.name: DeleteDepartment
-- piko.command: exec
DELETE FROM departments WHERE id = ?;

-- piko.name: CountEmployees
-- piko.command: one
SELECT COUNT(*) AS total FROM employees;
