-- piko.name: ListItemsWithFlags
-- piko.command: many
SELECT
    id,
    name,
    price > 10.0 AS is_expensive,
    category IS NOT NULL AS has_category,
    price BETWEEN 5.0 AND 20.0 AS in_price_range,
    stock > 0 AND category IS NOT NULL AS available_and_categorised
FROM items
ORDER BY id;
