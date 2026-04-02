-- piko.name: IncrementViews
-- piko.command: exec
UPDATE counters SET page_views = page_views + 1 WHERE id = ?;
