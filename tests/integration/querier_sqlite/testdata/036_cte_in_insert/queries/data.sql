-- piko.name: CopyFiltered
-- piko.command: many
WITH filtered AS (
    SELECT id, value FROM source_data WHERE length(value) > 3
)
INSERT INTO target_data (id, value) SELECT id, value FROM filtered RETURNING id, value;
