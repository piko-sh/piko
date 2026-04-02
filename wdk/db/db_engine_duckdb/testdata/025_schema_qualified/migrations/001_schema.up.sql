CREATE SCHEMA app;

CREATE TABLE app.accounts (
    id INTEGER PRIMARY KEY,
    username VARCHAR NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true
);
