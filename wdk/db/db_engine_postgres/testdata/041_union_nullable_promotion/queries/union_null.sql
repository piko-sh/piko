-- piko.name: UnionNull
-- piko.command: many
SELECT id, name, occurred_at FROM events
UNION ALL
SELECT 0 AS id, 'placeholder' AS name, NULL::timestamptz AS occurred_at;
