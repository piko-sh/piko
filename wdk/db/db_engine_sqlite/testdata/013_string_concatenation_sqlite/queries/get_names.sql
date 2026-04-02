-- piko.name: GetFullNames
-- piko.command: many
SELECT
  id,
  first_name || ' ' || last_name as full_name,
  first_name || ' ' || middle_name as with_middle
FROM people
