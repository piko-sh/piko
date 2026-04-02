-- piko.name: InsertItem
-- piko.command: one
INSERT INTO items (name, quantity, unit_price)
VALUES ($1, $2, $3)
RETURNING id, upper(name) AS upper_name, quantity * unit_price AS total, created_at;

-- piko.name: UpdateQuantity
-- piko.command: one
UPDATE items
SET quantity = quantity + $2
WHERE id = $1
RETURNING id, name, quantity AS new_quantity, quantity * unit_price AS new_total;

-- piko.name: DeleteItem
-- piko.command: one
DELETE FROM items
WHERE id = $1
RETURNING id, name, upper(name) || ' (DELETED)' AS deletion_label;
