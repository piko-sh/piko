CREATE TABLE documents (
    id SERIAL PRIMARY KEY,
    title TEXT NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'
);
