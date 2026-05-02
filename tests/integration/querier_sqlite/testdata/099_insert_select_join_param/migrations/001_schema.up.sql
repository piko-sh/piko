CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    active INTEGER NOT NULL
);

CREATE TABLE sessions (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL,
    session_token TEXT NOT NULL
);

CREATE TABLE sessions_archive (
    id INTEGER PRIMARY KEY,
    account_id INTEGER NOT NULL,
    session_token TEXT NOT NULL
);
