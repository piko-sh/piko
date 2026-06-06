CREATE TABLE accounts (
    id INTEGER PRIMARY KEY
);

CREATE TABLE account_versions (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL,
    email TEXT NOT NULL,
    status TEXT NOT NULL
);
