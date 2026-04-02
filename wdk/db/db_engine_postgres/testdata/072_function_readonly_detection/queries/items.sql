-- piko.name: ReadOnlyWithImmutable
-- piko.command: many
SELECT id, safe_multiply(quantity, 2) AS doubled FROM items;

-- piko.name: NotReadOnlyWithVolatile
-- piko.command: one
SELECT dangerous_update($1::integer) AS updated_id;

-- piko.name: NotReadOnlyWithUnknown
-- piko.command: one
SELECT some_unknown_function($1::integer) AS result;
