-- piko.query(name: GetMetadataField, command: one)
SELECT
    id,
    data ->> '$.name' AS extracted_name,
    json_extract(data, '$.count') AS extracted_count
FROM metadata
WHERE id = ?;
