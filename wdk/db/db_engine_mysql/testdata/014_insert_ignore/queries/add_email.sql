-- piko.query(name: AddEmail, command: exec)
INSERT IGNORE INTO unique_emails (email) VALUES (?);
