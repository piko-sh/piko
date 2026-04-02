CREATE TABLE counters (
    id SERIAL PRIMARY KEY,
    value INTEGER NOT NULL DEFAULT 0
);

CREATE FUNCTION pure_add(a INTEGER, b INTEGER) RETURNS INTEGER
LANGUAGE sql IMMUTABLE
AS $$ SELECT a + b; $$;

CREATE FUNCTION increment_counter(counter_id INTEGER) RETURNS VOID
LANGUAGE plpgsql VOLATILE
AS $$
BEGIN
    UPDATE counters SET value = value + 1 WHERE id = counter_id;
END;
$$;

CREATE FUNCTION wrapper_pure(x INTEGER) RETURNS INTEGER
LANGUAGE sql IMMUTABLE
AS $$ SELECT pure_add(x, 10); $$;

CREATE FUNCTION wrapper_dangerous(counter_id INTEGER) RETURNS INTEGER
LANGUAGE sql STABLE
AS $$ SELECT pure_add(counter_id, 1); $$;
