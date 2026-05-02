-- piko.query(name: ReadOnlyWithImmutable, command: many)
SELECT id, safe_multiply(quantity, 2) AS doubled FROM items;

-- piko.query(name: NotReadOnlyWithVolatile, command: one)
SELECT dangerous_update($1::integer) AS updated_id;

-- piko.query(name: NotReadOnlyWithUnknown, command: one)
SELECT some_unknown_function($1::integer) AS result;
