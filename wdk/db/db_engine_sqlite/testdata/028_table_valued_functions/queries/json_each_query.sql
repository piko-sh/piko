-- piko.name: ExtractKeys
-- piko.command: many
SELECT d.id, je.key, je.value
FROM documents d, json_each(d.metadata) je
