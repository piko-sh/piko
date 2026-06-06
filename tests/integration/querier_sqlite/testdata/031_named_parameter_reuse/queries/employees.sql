-- piko.query(name: FindByIDOrManager, command: many)
-- :target_id as piko.param
SELECT id, name, manager_id FROM employees WHERE id = :target_id OR manager_id = :target_id;
