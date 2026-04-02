-- piko.name: ExtractArrayItems
-- piko.command: many
SELECT d.id, elem.value AS item
FROM documents d, jsonb_array_elements(d.metadata) elem;
