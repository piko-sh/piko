CREATE TABLE items (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    quantity INTEGER NOT NULL
);

CREATE FUNCTION safe_multiply(a INTEGER, b INTEGER) RETURNS INTEGER
LANGUAGE sql IMMUTABLE
AS $$ SELECT a * b; $$;

CREATE FUNCTION dangerous_update(item_id INTEGER) RETURNS INTEGER
LANGUAGE plpgsql VOLATILE
AS $$
BEGIN
    UPDATE items SET quantity = quantity + 1 WHERE id = item_id;
    RETURN item_id;
END;
$$;
