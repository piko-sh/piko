-- piko.query(name: InsertDepartment, command: exec)
INSERT INTO departments (id, name) VALUES (?, ?);

-- piko.query(name: InsertEmployee, command: exec)
INSERT INTO employees (id, name, dept_id) VALUES (?, ?, ?);

-- piko.query(name: ListEmployees, command: many)
SELECT id, name, dept_id FROM employees ORDER BY id;

-- piko.query(name: DeleteDepartment, command: exec)
DELETE FROM departments WHERE id = ?;

-- piko.query(name: CountEmployees, command: one)
SELECT COUNT(*) AS total FROM employees;
