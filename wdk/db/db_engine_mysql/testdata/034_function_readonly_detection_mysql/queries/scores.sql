-- piko.name: ReadOnlyWithDeterministic
-- piko.command: many
SELECT id, safe_multiply(points, 2) AS doubled FROM scores;

-- piko.name: NotReadOnlyWithUnknown
-- piko.command: one
SELECT some_unknown_function(?) AS result;
