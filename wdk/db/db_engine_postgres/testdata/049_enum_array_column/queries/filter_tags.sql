-- piko.name: FilterTags
-- piko.command: many
SELECT id, name, tags FROM products WHERE tags @> $1::tag[];
