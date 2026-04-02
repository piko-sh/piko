CREATE TABLE audit_log (
    id INTEGER PRIMARY KEY,
    action TEXT NOT NULL,
    timestamp TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE accounts (
    id INTEGER PRIMARY KEY,
    name TEXT NOT NULL,
    balance REAL NOT NULL DEFAULT 0
);

CREATE TRIGGER accounts_audit_insert AFTER INSERT ON accounts
BEGIN
    INSERT INTO audit_log (action) VALUES ('insert:' || NEW.id);
END;

DROP TRIGGER IF EXISTS accounts_audit_insert;

CREATE TRIGGER accounts_audit_update AFTER UPDATE ON accounts
BEGIN
    INSERT INTO audit_log (action) VALUES ('update:' || NEW.id);
END;
