-- piko.name: ListAllWorkers
-- piko.command: many
SELECT id, name, department AS org, salary AS pay FROM employees WHERE department = ?
UNION ALL
SELECT id, name, agency AS org, rate AS pay FROM contractors WHERE agency = ?

-- piko.name: ListUniqueWorkerNames
-- piko.command: many
SELECT name FROM employees
UNION
SELECT name FROM contractors
ORDER BY name
