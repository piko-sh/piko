-- piko.name: ApplyPriceUpdates
-- piko.command: exec
UPDATE products
SET price = price_updates.new_price
FROM price_updates
WHERE products.id = price_updates.product_id;
