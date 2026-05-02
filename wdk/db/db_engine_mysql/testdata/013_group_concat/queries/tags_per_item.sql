-- piko.query(name: TagsPerItem, command: many)
SELECT
    item_id,
    GROUP_CONCAT(tag_name ORDER BY tag_name SEPARATOR ', ') AS all_tags
FROM tags
GROUP BY item_id;
