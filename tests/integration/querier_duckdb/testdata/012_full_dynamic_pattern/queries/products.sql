-- piko.name: BrowseProducts
-- piko.command: many
-- $1 as piko.optional(category)
-- $2 as piko.sortable(order_by) columns:name,price
SELECT id, name, price, category FROM products WHERE ($1 IS NULL OR category = $1)
