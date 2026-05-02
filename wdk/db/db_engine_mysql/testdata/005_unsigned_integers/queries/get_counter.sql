-- piko.query(name: GetCounter, command: one)
SELECT id, page_views, small_counter, tiny_flag, medium_counter
FROM counters
WHERE id = ?;
