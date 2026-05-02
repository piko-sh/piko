-- piko.query(name: SearchEmployees, command: many)
-- $1 as piko.param(department, optional: true)
-- $2 as piko.param(min_level, optional: true)
-- $3 as piko.param(active, optional: true)
SELECT id, name, department, level, active FROM employees WHERE ($1 IS NULL OR department = $1) AND ($2 IS NULL OR level >= $2) AND ($3 IS NULL OR active = $3) ORDER BY id ASC;
