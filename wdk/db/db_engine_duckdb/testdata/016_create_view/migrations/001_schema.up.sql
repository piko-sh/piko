CREATE TABLE users (
    id INTEGER PRIMARY KEY,
    name VARCHAR NOT NULL,
    email VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT current_timestamp
);

CREATE VIEW active_users AS
    SELECT id, name, email FROM users WHERE email IS NOT NULL;
