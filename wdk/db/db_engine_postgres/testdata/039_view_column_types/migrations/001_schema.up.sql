CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE VIEW active_users AS
    SELECT id, name, email FROM users WHERE email IS NOT NULL;
