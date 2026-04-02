-- piko.name: SetNestedField
-- piko.command: one
UPDATE documents
SET metadata = jsonb_set(metadata, '{status}', $2::jsonb)
WHERE id = $1
RETURNING id, title, metadata;

-- piko.name: MergeMetadata
-- piko.command: one
UPDATE documents
SET metadata = metadata || $2::jsonb
WHERE id = $1
RETURNING id, title, metadata;

-- piko.name: RemoveKey
-- piko.command: one
UPDATE documents
SET metadata = metadata - $2
WHERE id = $1
RETURNING id, title, metadata;
