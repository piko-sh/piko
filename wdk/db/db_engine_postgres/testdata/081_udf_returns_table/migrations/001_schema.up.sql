CREATE SCHEMA app;

CREATE TABLE app.users (
    id INTEGER PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL
);

CREATE OR REPLACE FUNCTION app.list_users(_min_id INTEGER DEFAULT NULL)
RETURNS TABLE (
    user_id INTEGER,
    user_email TEXT,
    user_created_at TIMESTAMPTZ
)
LANGUAGE sql STABLE
AS $$
    SELECT id, email, created_at
    FROM app.users
    WHERE id >= COALESCE(_min_id, 0);
$$;
