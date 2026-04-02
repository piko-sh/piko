-- piko.name: SearchUsers
-- piko.command: many
-- ?1 as piko.optional(name)
-- ?2 as piko.optional(role)
SELECT id, name, email, role FROM users WHERE (?1 IS NULL OR name = ?1) AND (?2 IS NULL OR role = ?2) ORDER BY id ASC;
