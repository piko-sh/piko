-- piko.query(name: GetAuditLog, command: many)
SELECT id, action, created_at FROM audit_log WHERE action = $1;
