-- piko.name: GetByActiveStatus
-- piko.command: many
-- ?1 as piko.param(is_active)
SELECT id, name FROM flags WHERE active = ?1 ORDER BY name;
