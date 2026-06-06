-- piko.query(name: BrowseProducts, command: many)
-- $1 as piko.param(category, optional: true)
-- piko.sortable(order_by, columns: [name, price])
SELECT id, name, price, category FROM products WHERE ($1 IS NULL OR category = $1)
