CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    balance INTEGER NOT NULL CHECK (balance >= 0),
    status TEXT NOT NULL CHECK (status IN ('active', 'inactive', 'suspended'))
);
