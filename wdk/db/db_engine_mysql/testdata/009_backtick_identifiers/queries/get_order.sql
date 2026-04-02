-- piko.name: GetOrder
-- piko.command: one
SELECT `id`, `select`, `from`, `date` FROM `order` WHERE `id` = ?;
