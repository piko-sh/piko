-- piko.name: GetItemsWithDefault
-- piko.command: many
-- ?1 as piko.param(default_priority)
SELECT id, name, priority FROM items ORDER BY priority = ?1 DESC, name;
