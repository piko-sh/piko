-- piko.name: AggTags
-- piko.command: many
SELECT group_id, string_agg(name, ', ' ORDER BY name) AS tag_list
FROM tags
GROUP BY group_id;
