-- piko.query(name: GetOrder, command: one)
SELECT `id`, `select`, `from`, `date` FROM `order` WHERE `id` = ?;
