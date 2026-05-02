-- piko.query(name: ValuesCTE, command: many)
WITH statuses AS (VALUES ('active'), ('inactive'), ('pending'))
SELECT column1 AS status FROM statuses;
