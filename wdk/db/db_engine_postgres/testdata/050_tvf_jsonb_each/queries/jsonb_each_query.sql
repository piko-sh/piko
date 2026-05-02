-- piko.query(name: ExtractKeyValues, command: many)
SELECT d.id, d.title, kv.key, kv.value
FROM documents d, jsonb_each(d.metadata) kv;
