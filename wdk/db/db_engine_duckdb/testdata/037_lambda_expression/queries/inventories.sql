-- piko.name: GetDoubledQuantities
-- piko.command: many
SELECT id, name, list_transform(quantities, x -> x * 2) AS doubled FROM inventories;

-- piko.name: GetPositiveQuantities
-- piko.command: many
SELECT id, name, list_filter(quantities, x -> x > 0) AS positive FROM inventories;
