CREATE TABLE audit_log (
    id SERIAL PRIMARY KEY,
    action TEXT NOT NULL,
    detail TEXT
);
