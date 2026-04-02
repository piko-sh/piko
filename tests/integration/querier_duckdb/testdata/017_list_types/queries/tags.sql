-- piko.name: GetTagsData
-- piko.command: one
SELECT id, name, CAST(tags AS VARCHAR) AS tags FROM tags_data WHERE id = $1;

-- piko.name: CountTags
-- piko.command: one
SELECT id, array_length(tags) AS tag_count FROM tags_data WHERE id = $1;
