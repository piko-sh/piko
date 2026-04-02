CREATE TABLE accounts (
    id SERIAL PRIMARY KEY,
    balance NUMERIC(10,2) NOT NULL DEFAULT 0
);
CREATE TABLE transactions (
    id SERIAL PRIMARY KEY,
    account_id INTEGER NOT NULL REFERENCES accounts(id),
    amount NUMERIC(10,2) NOT NULL
);
