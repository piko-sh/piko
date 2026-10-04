-- piko.query(name: ListSlugs, command: many)
SELECT slug FROM events
UNION ALL
SELECT slug FROM events
UNION ALL
SELECT NULL;
