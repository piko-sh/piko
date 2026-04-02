CREATE SCHEMA base;

CREATE TABLE base.entity (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE events (
    name TEXT NOT NULL,
    payload JSONB
) INHERITS (base.entity);
