CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE tokens (
    id UUID NOT NULL DEFAULT gen_random_uuid(),
    value TEXT NOT NULL
);
