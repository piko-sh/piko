-- piko.name: GetItemTags
-- piko.command: many
SELECT item_id, json_group_array(label) AS tag_list FROM tags GROUP BY item_id ORDER BY item_id;
