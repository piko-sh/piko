-- piko.name: CountUnion
-- piko.command: one
WITH t AS (
    SELECT thing FROM things
    UNION ALL
    SELECT thing FROM things
)
SELECT COUNT(*) AS total FROM t;
