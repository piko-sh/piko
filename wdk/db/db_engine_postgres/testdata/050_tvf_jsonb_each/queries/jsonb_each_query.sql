-- piko.name: ExtractKeyValues
-- piko.command: many
SELECT d.id, d.title, kv.key, kv.value
FROM documents d, jsonb_each(d.metadata) kv;
