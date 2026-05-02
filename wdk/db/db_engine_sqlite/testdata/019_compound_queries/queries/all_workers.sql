-- piko.query(name: ListAllWorkers, command: many)
SELECT id, name, department AS org, salary AS pay FROM employees WHERE department = ?
UNION ALL
SELECT id, name, agency AS org, rate AS pay FROM contractors WHERE agency = ?

-- piko.query(name: ListUniqueWorkerNames, command: many)
SELECT name FROM employees
UNION
SELECT name FROM contractors
ORDER BY name
