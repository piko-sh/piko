-- piko.name: TagsByCategory
-- piko.command: many
SELECT category_id, string_agg(name, ', ') AS tag_list FROM tags GROUP BY category_id;
