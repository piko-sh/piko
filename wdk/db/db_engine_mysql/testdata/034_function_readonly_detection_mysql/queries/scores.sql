-- piko.query(name: ReadOnlyWithDeterministic, command: many)
SELECT id, safe_multiply(points, 2) AS doubled FROM scores;

-- piko.query(name: NotReadOnlyWithUnknown, command: one)
SELECT some_unknown_function(?) AS result;
