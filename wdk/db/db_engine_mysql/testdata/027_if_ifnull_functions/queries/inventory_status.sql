-- piko.name: InventoryStatus
-- piko.command: many
SELECT
    id,
    item_name,
    quantity,
    IF(quantity < 10, 'low', 'ok') AS stock_status,
    IFNULL(reorder_level, 0) AS effective_reorder
FROM inventory;
