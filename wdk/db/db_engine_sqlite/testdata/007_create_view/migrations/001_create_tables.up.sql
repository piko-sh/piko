CREATE TABLE employees (
  id INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  department TEXT NOT NULL,
  salary REAL NOT NULL
);

CREATE VIEW active_employees (id, name, department, salary) AS
SELECT id, name, department, salary FROM employees;
