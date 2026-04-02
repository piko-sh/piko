CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL
);

CREATE FUNCTION volatile_func(val INTEGER) RETURNS INTEGER
LANGUAGE sql VOLATILE
AS $$ SELECT val * 2 $$;

-- piko.readonly
CREATE FUNCTION overridden_readonly_func(val INTEGER) RETURNS INTEGER
LANGUAGE sql VOLATILE
AS $$ SELECT val * 2 $$;

-- piko.readonly(false)
CREATE FUNCTION overridden_not_readonly_func(val INTEGER) RETURNS INTEGER
LANGUAGE sql IMMUTABLE
AS $$ SELECT val * 2 $$;
