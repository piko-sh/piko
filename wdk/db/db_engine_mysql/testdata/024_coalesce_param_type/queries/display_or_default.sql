-- piko.query(name: DisplayOrDefault, command: many)
SELECT id, COALESCE(display_name, ?) AS name FROM profiles;
