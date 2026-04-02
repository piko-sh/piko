CREATE TABLE settings (
    id SERIAL PRIMARY KEY,
    value TEXT,
    fallback TEXT NOT NULL DEFAULT 'none'
);
