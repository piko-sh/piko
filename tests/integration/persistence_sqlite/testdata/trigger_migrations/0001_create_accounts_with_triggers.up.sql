CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE account_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id TEXT NOT NULL,
    action TEXT NOT NULL,
    at TEXT NOT NULL DEFAULT (datetime('now'))
);


CREATE TRIGGER tr_accounts_disallow_update
    BEFORE UPDATE ON accounts
    FOR EACH ROW
BEGIN
    SELECT RAISE(ABORT, 'Cannot UPDATE on accounts');
END;


CREATE TRIGGER tr_accounts_log_insert
    AFTER INSERT ON accounts
    FOR EACH ROW
BEGIN
    INSERT INTO account_log (account_id, action) VALUES (NEW.id, 'created');
    INSERT INTO account_log (account_id, action) VALUES (NEW.id, 'audited');
END;
