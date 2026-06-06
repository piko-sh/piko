-- piko.query(name: SetNestedField, command: one)
UPDATE documents
SET metadata = jsonb_set(metadata, '{status}', $2::jsonb)
WHERE id = $1
RETURNING id, title, metadata;

-- piko.query(name: MergeMetadata, command: one)
UPDATE documents
SET metadata = metadata || $2::jsonb
WHERE id = $1
RETURNING id, title, metadata;

-- piko.query(name: RemoveKey, command: one)
UPDATE documents
SET metadata = metadata - $2
WHERE id = $1
RETURNING id, title, metadata;
