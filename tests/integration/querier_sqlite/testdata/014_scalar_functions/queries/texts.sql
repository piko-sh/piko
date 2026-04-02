-- piko.name: GetTextTransforms
-- piko.command: one
SELECT
    id,
    lower(content) AS lowered,
    upper(content) AS uppered,
    length(content) AS content_length,
    substr(content, 1, 5) AS prefix,
    lower(optional_content) AS optional_lowered,
    length(optional_content) AS optional_length
FROM texts
WHERE id = ?;
