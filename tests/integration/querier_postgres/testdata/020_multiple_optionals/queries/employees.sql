-- piko.name: SearchEmployees
-- piko.command: many
-- $1 as piko.optional(department)
-- $2 as piko.optional(min_level)
-- $3 as piko.optional(active)
SELECT id, name, department, level, active FROM employees WHERE ($1 IS NULL OR department = $1) AND ($2 IS NULL OR level >= $2) AND ($3 IS NULL OR active = $3) ORDER BY id ASC;
