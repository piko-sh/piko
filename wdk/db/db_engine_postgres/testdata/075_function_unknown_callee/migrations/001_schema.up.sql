CREATE TABLE records (
    id SERIAL PRIMARY KEY,
    data TEXT NOT NULL
);

CREATE FUNCTION calls_unknown() RETURNS INTEGER
LANGUAGE sql STABLE
AS $$ SELECT some_extension_function(42); $$;

CREATE FUNCTION calls_known_pure(val INTEGER) RETURNS INTEGER
LANGUAGE sql IMMUTABLE
AS $$ SELECT val + 1; $$;
