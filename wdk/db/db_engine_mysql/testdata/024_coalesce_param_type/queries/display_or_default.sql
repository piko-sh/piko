-- piko.name: DisplayOrDefault
-- piko.command: many
SELECT id, COALESCE(display_name, ?) AS name FROM profiles;
