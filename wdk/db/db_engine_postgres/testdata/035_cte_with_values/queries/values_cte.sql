-- piko.name: ValuesCTE
-- piko.command: many
WITH statuses AS (VALUES ('active'), ('inactive'), ('pending'))
SELECT column1 AS status FROM statuses;
