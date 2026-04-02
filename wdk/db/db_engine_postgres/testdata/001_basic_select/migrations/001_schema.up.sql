CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email VARCHAR(255),
    age INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
