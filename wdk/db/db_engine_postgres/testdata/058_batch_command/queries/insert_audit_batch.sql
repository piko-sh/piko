-- piko.name: InsertAuditBatch
-- piko.command: batch
INSERT INTO audit_log (action, detail) VALUES ($1, $2);
