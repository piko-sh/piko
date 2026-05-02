-- piko.query(name: InsertAuditBatch, command: batch)
INSERT INTO audit_log (action, detail) VALUES ($1, $2);
