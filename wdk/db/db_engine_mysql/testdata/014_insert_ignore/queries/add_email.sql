-- piko.name: AddEmail
-- piko.command: exec
INSERT IGNORE INTO unique_emails (email) VALUES (?);
